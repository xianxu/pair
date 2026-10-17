"""Pure source analysis for the supported Python Pair subprocess test recipes.

Proof starts only at the shared helper's --print0 output. Supported consumers
are its dictionary comprehension, dictionary copies/aliases, and updates whose
positional mappings are already proven and whose keyword entries are explicit.
Unknown mapping expressions fail closed. Statements run in order, function
locals start fresh, and branch/loop joins retain proof only on all paths. This
is a regression guard, not a general Python interpreter or security sandbox.
"""
import ast
import json
import pathlib
import sys

UNKNOWN, HELPER, RAW, ENV, PAIR = range(5)
SPAWNS = {'run', 'call', 'check_call', 'check_output', 'Popen'}


class Analyzer:
    def __init__(self, source):
        self.tree = ast.parse(source)
        self.imports = {'subprocess': 'subprocess', 'os': 'os'}
        self.failures = []
        for node in ast.walk(self.tree):
            if isinstance(node, ast.Import):
                for item in node.names:
                    self.imports[item.asname or item.name] = item.name
            elif isinstance(node, ast.ImportFrom):
                for item in node.names:
                    self.imports[item.asname or item.name] = (node.module or '') + '.' + item.name

    def name(self, node):
        if isinstance(node, ast.Name):
            return self.imports.get(node.id, node.id)
        if isinstance(node, ast.Attribute):
            return self.name(node.value) + '.' + node.attr
        return ''

    def value(self, node, facts):
        if isinstance(node, ast.Name):
            return facts.get(node.id, UNKNOWN)
        if isinstance(node, ast.Constant) and isinstance(node.value, str):
            base = pathlib.PurePosixPath(node.value).name
            if base == 'pair-test-env.sh':
                return HELPER
            if base in ('pair', 'pair-go', 'pair-wrap') or (base.startswith('pair-') and not base.endswith(('.sh', '.py'))):
                return PAIR
        if isinstance(node, ast.BinOp):
            kinds = (self.value(node.left, facts), self.value(node.right, facts))
            return HELPER if HELPER in kinds else PAIR if PAIR in kinds else UNKNOWN
        if not isinstance(node, ast.Call):
            return UNKNOWN
        name = self.name(node.func)
        if name in ('str', 'Path', 'pathlib.Path') and node.args:
            return self.value(node.args[0], facts)
        if name in ('os.getenv', 'os.environ.get') and node.args and isinstance(node.args[0], ast.Constant):
            return PAIR if node.args[0].value in ('PAIR_NATIVE_BINARY', 'PAIR_TEST_BINARY', 'PAIR_BIN') else UNKNOWN
        if name == 'subprocess.check_output' and node.args and isinstance(node.args[0], (ast.List, ast.Tuple)):
            argv = node.args[0].elts
            if (len(argv) >= 4 and isinstance(argv[0], ast.Constant) and argv[0].value in ('/bin/sh', 'sh')
                    and self.value(argv[1], facts) == HELPER and isinstance(argv[2], ast.Constant) and argv[2].value == '--print0'):
                return RAW
        if name == 'dict':
            if len(node.args) == 1 and isinstance(node.args[0], ast.GeneratorExp) and not node.keywords:
                generator = node.args[0]
                if len(generator.generators) == 1:
                    iterator = generator.generators[0].iter
                    if (isinstance(iterator, ast.Call) and isinstance(iterator.func, ast.Attribute)
                            and iterator.func.attr == 'split' and self.value(iterator.func.value, facts) == RAW
                            and not any(self.name(n) == 'os.environ' for n in ast.walk(generator.elt))):
                        return ENV
            if node.args and all(self.value(arg, facts) == ENV for arg in node.args) and all(
                    kw.arg is not None or self.value(kw.value, facts) == ENV for kw in node.keywords):
                return ENV
        if isinstance(node.func, ast.Attribute) and node.func.attr == 'copy' and self.value(node.func.value, facts) == ENV:
            return ENV
        return UNKNOWN

    def effects(self, node, facts):
        if node is None:
            return
        # ast.walk over a function body would merge scopes; statement traversal
        # owns those separately. Here only expression nodes are supplied.
        for call in ast.walk(node):
            if not isinstance(call, ast.Call):
                continue
            name = self.name(call.func)
            if name.startswith('subprocess.') and name.split('.')[-1] in SPAWNS and call.args:
                argv = call.args[0]
                if isinstance(argv, (ast.List, ast.Tuple)) and argv.elts:
                    first = argv.elts[0]
                    pair = self.value(first, facts) == PAIR or (isinstance(first, ast.Name) and first.id in ('binary', 'pair', 'pair_bin'))
                    if pair:
                        env = next((kw.value for kw in call.keywords if kw.arg == 'env'), None)
                        if self.value(env, facts) != ENV:
                            self.failures.append(call.lineno)
            if isinstance(call.func, ast.Attribute) and self.value(call.func.value, facts) == ENV:
                method = call.func.attr
                if method == 'update':
                    safe = all(self.value(arg, facts) == ENV for arg in call.args) and all(
                        kw.arg is not None or self.value(kw.value, facts) == ENV for kw in call.keywords)
                else:
                    safe = method in ('copy', 'get', 'items', 'keys', 'values')
                if not safe:
                    # Conservatively invalidate every mapping proof, including
                    # aliases, rather than pretending Python dicts are values.
                    for key in list(facts):
                        if facts[key] == ENV:
                            facts[key] = UNKNOWN

    @staticmethod
    def join(paths):
        keys = set().union(*(path.keys() for path in paths))
        return {key: paths[0].get(key, UNKNOWN) if all(path.get(key, UNKNOWN) == paths[0].get(key, UNKNOWN) for path in paths) else UNKNOWN for key in keys}

    def block(self, statements, facts):
        facts = dict(facts)
        for stmt in statements:
            if isinstance(stmt, (ast.FunctionDef, ast.AsyncFunctionDef)):
                # Module constants may identify the helper; local mapping proof
                # never crosses a function boundary through a reused name.
                local = {k: v for k, v in facts.items() if v == HELPER}
                args = [*stmt.args.posonlyargs, *stmt.args.args, *stmt.args.kwonlyargs]
                for arg in [*args, stmt.args.vararg, stmt.args.kwarg]:
                    if arg is not None:
                        local[arg.arg] = UNKNOWN
                self.block(stmt.body, local)
                facts[stmt.name] = UNKNOWN
            elif isinstance(stmt, ast.ClassDef):
                self.block(stmt.body, {})
            elif isinstance(stmt, (ast.Assign, ast.AnnAssign, ast.NamedExpr)):
                value = stmt.value
                self.effects(value, facts)
                origin = self.value(value, facts)
                targets = stmt.targets if isinstance(stmt, ast.Assign) else [stmt.target]
                for target in targets:
                    if isinstance(target, ast.Name):
                        facts[target.id] = origin
                    elif isinstance(target, (ast.Tuple, ast.List)):
                        for item in target.elts:
                            if isinstance(item, ast.Name):
                                facts[item.id] = UNKNOWN
            elif isinstance(stmt, ast.If):
                self.effects(stmt.test, facts)
                facts = self.join([self.block(stmt.body, facts), self.block(stmt.orelse, facts)])
            elif isinstance(stmt, (ast.For, ast.AsyncFor, ast.While)):
                self.effects(stmt.iter if isinstance(stmt, (ast.For, ast.AsyncFor)) else stmt.test, facts)
                # A loop may execute zero times. Its body cannot establish proof
                # for a later spawn unless the pre-loop state already has it.
                entered = dict(facts)
                if isinstance(stmt, (ast.For, ast.AsyncFor)):
                    for name in ast.walk(stmt.target):
                        if isinstance(name, ast.Name):
                            entered[name.id] = UNKNOWN
                facts = self.join([facts, self.block(stmt.body, entered)])
                facts = self.block(stmt.orelse, facts)
            elif isinstance(stmt, (ast.With, ast.AsyncWith)):
                for item in stmt.items:
                    self.effects(item.context_expr, facts)
                    if item.optional_vars is not None:
                        for name in ast.walk(item.optional_vars):
                            if isinstance(name, ast.Name):
                                facts[name.id] = UNKNOWN
                facts = self.block(stmt.body, facts)
            elif isinstance(stmt, ast.Try):
                normal = self.block(stmt.orelse, self.block(stmt.body, facts))
                paths = [normal]
                for handler in stmt.handlers:
                    # An exception may occur after any prefix of the try body.
                    # No pre-try proof survives without explicit reconstruction.
                    entered = {key: UNKNOWN for key in facts}
                    if handler.name:
                        entered[handler.name] = UNKNOWN
                    paths.append(self.block(handler.body, entered))
                facts = self.join(paths)
                if stmt.finalbody:
                    # finally also runs after unhandled, mid-body exceptions.
                    facts = self.block(stmt.finalbody, {key: UNKNOWN for key in facts})
            elif isinstance(stmt, (ast.Expr, ast.Return, ast.Raise, ast.Assert)):
                self.effects(stmt.value if hasattr(stmt, 'value') else stmt.test if isinstance(stmt, ast.Assert) else stmt.exc, facts)
            elif isinstance(stmt, (ast.Import, ast.ImportFrom, ast.Pass, ast.Break, ast.Continue)):
                pass
            else:
                # Unsupported control/dataflow cannot introduce or preserve a
                # mapping proof. Never bless a source by its spelling alone.
                for key in list(facts):
                    if facts[key] == ENV:
                        facts[key] = UNKNOWN
                self.effects(stmt, facts)
        return facts

    def analyze(self):
        self.block(self.tree.body, {})
        return sorted(set(self.failures))


if __name__ == '__main__':
    try:
        print(json.dumps(Analyzer(sys.stdin.read()).analyze()))
    except (SyntaxError, ValueError) as error:
        print(str(error), file=sys.stderr)
        raise SystemExit(2)
