"""Strict native evidence must prove execution, completion and package identity."""
import importlib.util
import json
from pathlib import Path
import unittest
from unittest.mock import patch


def load_validator():
    path = Path(__file__).with_name("conformance-vm.py")
    if not path.exists():
        return None
    spec = importlib.util.spec_from_file_location("conformance_vm", path)
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


class ResultsTest(unittest.TestCase):
    def setUp(self):
        self.vm = load_validator()
        self.assertIsNotNone(self.vm, "pure conformance validator is missing")
        self.required = {"example/native": {"TestNative", "TestNative/child"}}

    def events(self):
        p = "example/native"
        return [dict(Action="start", Package=p),
                dict(Action="run", Package=p, Test="TestNative"),
                dict(Action="run", Package=p, Test="TestNative/child"),
                dict(Action="pass", Package=p, Test="TestNative/child", Elapsed=0.01),
                dict(Action="pass", Package=p, Test="TestNative", Elapsed=0.02),
                dict(Action="pass", Package=p, Elapsed=0.03)]

    def validate(self, events, required=None):
        return self.vm.validate_results((json.dumps(e) + "\n" for e in events),
                                        self.required if required is None else required)

    def rejects(self, events):
        with self.assertRaises((ValueError, RuntimeError)):
            self.validate(events)

    def test_go_artifact_and_attribute_metadata_is_not_completion(self):
        for metadata in [dict(Action="artifacts", Path="/guest/artifact"),
                         dict(Action="attr", Key="case", Value="native")]:
            with self.subTest(metadata=metadata):
                event = dict(metadata, Package="example/native", Test="TestNative")
                events = self.events()
                events.insert(2, event)
                self.validate(events)
                self.rejects([events[0], events[1], event, events[-1]])
                for key in ('Path',) if metadata['Action'] == 'artifacts' else ('Key', 'Value'):
                    bad = dict(event, **{key: 1})
                    events[2] = bad
                    self.rejects(events)
                    events[2] = dict(event)
                    del events[2][key]
                    if key == 'Path':
                        self.rejects(events)
                    else:
                        self.validate(events)  # Go omits empty attr strings.
                    events[2] = event

    def test_complete_summary(self):
        result = self.validate(self.events())
        self.assertEqual(result["events"], 6)
        self.assertEqual(set(result["passed"]["example/native"]), self.required["example/native"])

    def test_all_truncated_prefixes_refuse(self):
        for n in range(len(self.events())):
            with self.subTest(prefix=n):
                self.rejects(self.events()[:n])

    def test_removing_each_execution_or_terminal_event_refuses(self):
        for n in range(len(self.events())):
            with self.subTest(removed=n):
                events = self.events()
                self.rejects(events[:n] + events[n + 1:])

    def test_package_only_success_refuses(self):
        self.rejects([self.events()[0], self.events()[-1]])

    def test_identity_cannot_be_borrowed_from_another_package(self):
        events = self.events()
        events[3]["Package"] = "example/other"
        self.rejects(events)

    def test_duplicate_terminal_and_run_and_package_start_refuse(self):
        for index in (0, 1, 3, 5):
            with self.subTest(index=index):
                events = self.events()
                self.rejects(events[:index] + [events[index]] + events[index:])

    def test_failure_or_skip_anywhere_refuses(self):
        for action in ("fail", "skip", "build-fail"):
            with self.subTest(action=action):
                self.rejects(self.events() + [dict(Action=action, Package="example/native",
                                                 ImportPath="example/dependency")])

    def test_known_types_and_unknown_actions_refuse(self):
        for key, value in [("Action", 3), ("Action", "future-action"), ("Package", None),
                           ("Test", []), ("Output", {}), ("Elapsed", True),
                           ("Elapsed", -1), ("Elapsed", float("inf")),
                           ("Time", 2), ("OutputType", False), ("FailedBuild", None),
                           ("ImportPath", 8)]:
            with self.subTest(key=key, value=value):
                events = self.events()
                events[2][key] = value
                self.rejects(events)

    def test_bad_json_objects_and_duplicate_keys_refuse(self):
        for line in ('', 'null', '[]', '42', '{', '{}',
                     '{"Action":"start","Action":"pass","Package":"example/native"}',
                     '{"Action":"start","Package":"example/native","future":{"x":1,"x":2}}',
                     '{"Action":"start","Package":"example/native","future":NaN}'):
            with self.subTest(line=line), self.assertRaises((RuntimeError, ValueError)):
                self.vm.validate_results([line], self.required)

    def test_output_pause_cont_build_and_additive_fields(self):
        events = self.events()
        events[0]["future"] = {"nested": [1, True, None]}
        events[2:2] = [dict(Action="pause", Package="example/native", Test="TestNative"),
                       dict(Action="output", Package="example/native", Test="TestNative",
                            Output="=== PAUSE TestNative\n", OutputType="frame"),
                       dict(Action="cont", Package="example/native", Test="TestNative")]
        events.insert(0, dict(Action="build-output", ImportPath="example/dependency",
                              Output="build diagnostic\n"))
        result = self.validate(events)
        self.assertEqual(result["events"], len(events))

    def test_package_output_after_test_pass_is_legitimate(self):
        events = self.events()
        events.insert(-1, dict(Action="output", Package="example/native",
                               Test="TestNative", Output="--- PASS: TestNative\n"))
        self.validate(events)

    def test_invalid_pause_cont_and_post_package_events_refuse(self):
        for action in ("pause", "cont", "output", "run"):
            with self.subTest(action=action):
                self.rejects(self.events() + [dict(Action=action, Package="example/native",
                                                  Test="TestNative", Output="late")])
        events = self.events()
        events.insert(2, dict(Action="cont", Package="example/native", Test="TestNative"))
        self.rejects(events)

    def test_package_pass_cannot_hide_unfinished_extra_test(self):
        events = self.events()
        events.insert(-1, dict(Action="run", Package="example/native", Test="TestExtra"))
        self.rejects(events)

    def test_elapsed_outside_go_float64_range_refuses_cleanly(self):
        for elapsed in (10 ** 400, -(10 ** 400)):
            events = self.events()
            events[-1]["Elapsed"] = elapsed
            self.rejects(events)

    def test_empty_or_malformed_required_mapping_refuses(self):
        for required in ({}, {"example/native": set()}, {"": {"TestNative"}},
                         {"example/native": "TestNative"}, {"example/native": {3}},
                         {"example/native": None}):
            with self.subTest(required=required), self.assertRaises((RuntimeError, ValueError)):
                self.validate(self.events(), required)

    def test_invalid_transition_and_malformed_build_output_refuse(self):
        for injected in (dict(Action="pause", Package="example/native", Test="NotStarted"),
                         dict(Action="pass", Package="example/native", Test="NotStarted"),
                         dict(Action="run", Package="example/native"),
                         dict(Action="build-output", ImportPath="example/dependency"),
                         dict(Action="build-output", ImportPath="", Output="diagnostic"),
                         dict(Action="output", Package="example/native")):
            with self.subTest(injected=injected):
                events = self.events()
                events.insert(1, injected)
                self.rejects(events)

    def test_interleaved_packages(self):
        a = self.events()
        b = [dict(e, Package="example/other") for e in a]
        events = [e for pair in zip(a, b) for e in pair]
        result = self.validate(events, dict(self.required, **{"example/other": self.required["example/native"]}))
        self.assertEqual(set(result["passed"]), {"example/native", "example/other"})

    def test_one_iteration_and_one_parse_per_line_at_n_and_2n(self):
        for n in (50, 100):
            events = [self.events()[0]]
            for i in range(n):
                events += [dict(Action="run", Package="example/native", Test=f"Test{i}"),
                           dict(Action="pass", Package="example/native", Test=f"Test{i}")]
            events.append(self.events()[-1])
            class Counted:
                iterations = 0
                yielded = 0
                def __iter__(self):
                    self.iterations += 1
                    if self.iterations != 1:
                        raise AssertionError("evidence iterated twice")
                    for e in events:
                        self.yielded += 1
                        yield json.dumps(e) + "\n"
            lines = Counted()
            with patch.object(self.vm.json, "loads", wraps=json.loads) as loads:
                self.vm.validate_results(lines, {"example/native": {f"Test{i}" for i in range(n)}})
            self.assertEqual((lines.iterations, lines.yielded, loads.call_count), (1, len(events), len(events)))


if __name__ == "__main__":
    unittest.main()
