"""Diagnostics at the process fixture's wanted-observation boundary."""
import json
import subprocess
import time

IDENTITY_FIELDS = ('status', 'repo', 'branch', 'head', 'file', 'diagnostic')


def identity_metadata(value):
    """Keep identity evidence without snapshot, body, token or document bytes."""
    if not isinstance(value, dict):
        return {'type': type(value).__name__}
    return {key: (value[key][:2048] if isinstance(value.get(key), str)
                  else {'type': type(value[key]).__name__})
            for key in IDENTITY_FIELDS if key in value}


def identity_problem(value):
    if not isinstance(value, dict):
        return 'resolver did not return a JSON object'
    if value.get('status') != 'resolved':
        return 'resolver status is not resolved'
    missing = [key for key in ('repo', 'branch', 'head', 'file')
               if not isinstance(value.get(key), str) or not value[key]]
    return 'incomplete resolved identity: ' + ', '.join(missing) if missing else None


def observe_wanted(command, env, history, phase, timeout=4, runner=None):
    # Only the resolver's positional directory belongs in the diagnostic.
    # Arbitrary executable arguments can themselves contain document bytes.
    args = command[:5] if command[1:4] == ['review', 'readiness', '--resolve'] else command[:1]
    record = dict(order=len(history) + 1, phase=phase,
                  command=[str(arg)[:2048] for arg in args],
                  started_ns=time.monotonic_ns(), returncode=None, timeout=timeout)
    history.append(record)
    value = None
    try:
        result = (runner or subprocess.run)(command, env=env, text=True,
                                            capture_output=True, timeout=timeout)
        record.update(returncode=result.returncode, stderr=result.stderr[-4096:],
                      stdout_bytes=len(result.stdout.encode()))
        try:
            value = json.loads(result.stdout)
            record['identity'] = identity_metadata(value)
        except (ValueError, TypeError) as error:
            record['validation_error'] = 'malformed JSON: ' + str(error)
        if result.returncode:
            record['validation_error'] = 'resolver exited with ' + str(result.returncode)
        elif 'validation_error' not in record:
            problem = identity_problem(value)
            if problem:
                record['validation_error'] = problem
    except subprocess.TimeoutExpired:
        record['validation_error'] = 'resolver process timed out'
    except OSError as error:
        record['validation_error'] = 'resolver execution failed: ' + str(error.strerror)
    finally:
        record['completed_ns'] = time.monotonic_ns()
    if 'validation_error' in record:
        raise AssertionError('wanted observation refused before forwarding: ' +
                             json.dumps(record, sort_keys=True))
    return value
