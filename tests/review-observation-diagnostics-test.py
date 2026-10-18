"""The observation fixture must retain the evidence its controller compares."""
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(ROOT / 'tests/lib'))
from review_observation_diagnostics import observe_wanted
from review_test_env import ReviewTestEnvironment


class WantedObservationTest(unittest.TestCase):
    def command(self, value, code=0, stderr=''):
        return [sys.executable, '-c',
                'import sys;sys.stdout.write(' + repr(json.dumps(value)) + ');'
                'sys.stderr.write(' + repr(stderr) + ');sys.exit(' + str(code) + ')']

    def test_exit_zero_invalid_wanted_is_rejected_with_original_diagnostic(self):
        history = []
        invalid = dict(status='invalid', repo='/fixture', branch='', head='', file='',
                       diagnostic='cannot resolve branch: context deadline exceeded',
                       snapshot='DO NOT RETAIN DOCUMENT BYTES')
        with self.assertRaises(AssertionError) as failure:
            observe_wanted(self.command(invalid), os.environ, history, 'wanted:restore-b')
        self.assertIn('context deadline exceeded', str(failure.exception))
        self.assertEqual(history[-1]['returncode'], 0)
        self.assertEqual(history[-1]['phase'], 'wanted:restore-b')
        self.assertEqual(history[-1]['identity']['status'], 'invalid')
        self.assertNotIn(invalid['snapshot'], str(failure.exception))

    def test_resolved_identity_is_forwarded_intact_and_trace_is_ordered(self):
        history = []
        for ordinal, branch in enumerate(('review/b', 'review/a'), 1):
            wanted = dict(status='resolved', repo='/fixture', branch=branch,
                          head='fixture-head', file='a.md', snapshot='PRIVATE SNAPSHOT')
            self.assertEqual(observe_wanted(self.command(wanted), os.environ, history,
                                            'wanted:' + branch), wanted)
            self.assertEqual(history[-1]['order'], ordinal)
            self.assertEqual(history[-1]['identity']['branch'], branch)
        self.assertNotIn('PRIVATE SNAPSHOT', json.dumps(history))

    def test_incomplete_nonzero_and_malformed_observations_name_their_cause(self):
        cases = [
            (dict(status='resolved', repo='/fixture', branch='review/a', head='h', file=''),
             0, '', 'file'),
            ([], 0, '', 'JSON object'),
            (dict(status='resolved'), 37, 'child failed', 'child failed'),
        ]
        for value, code, stderr, reason in cases:
            with self.subTest(value=value, code=code):
                history = []
                with self.assertRaises(AssertionError) as failure:
                    observe_wanted(self.command(value, code, stderr), os.environ,
                                    history, 'wanted:controlled')
                self.assertIn(reason, str(failure.exception))
                self.assertEqual(history[-1]['returncode'], code)
        history = []
        with self.assertRaises(AssertionError) as failure:
            observe_wanted([sys.executable, '-c', 'print("not JSON")'], os.environ,
                            history, 'wanted:malformed')
        self.assertIn('malformed JSON', str(failure.exception))
        self.assertEqual(history[-1]['stdout_bytes'], len('not JSON\n'))

    def test_process_timeout_keeps_phase_without_copying_command_payload(self):
        history = []
        secret = 'DOCUMENT_BYTES_IN_COMMAND'
        command = [sys.executable, '-c', 'import signal;value=' + repr(secret) + ';signal.pause()']
        # An already-expired process budget forces the timeout path while the
        # child cannot naturally finish. subprocess.run owns killing and join.
        with self.assertRaises(AssertionError) as failure:
            observe_wanted(command, os.environ, history, 'wanted:expired', timeout=0)
        self.assertNotIn(secret, str(failure.exception))
        self.assertEqual(history[-1]['phase'], 'wanted:expired')
        self.assertIn('timed out', history[-1]['validation_error'])


class ObservationFixtureDiagnosticsTest(unittest.TestCase):
    def test_positive_control_retains_wanted_fresh_and_request_order(self):
        with tempfile.TemporaryDirectory(prefix='pair-observation-positive-') as storage:
            report = Path(storage) / 'report.json'
            result = subprocess.run(['bash', str(ROOT / 'tests/review-observation-test.sh')],
                                    env=dict(os.environ, PAIR_TEST_OBSERVATION_REPORT=str(report)),
                                    text=True, capture_output=True)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            self.assertTrue(report.exists(), 'positive fixture discarded its observation evidence')
            evidence = json.loads(report.read_text())
            self.assertEqual(evidence['outcome'], 'passed')
            self.assertNotIn('"snapshot":', json.dumps(evidence))
            self.assertNotIn('"latest_agent_body":', json.dumps(evidence))
            self.assertEqual(len(evidence['wanted']), 5)
            self.assertIn('phase_evidence', evidence)
            phases = evidence['phase_evidence']
            positive = [event for event in phases if event['phase'] == 'slow-same-branch-observation']
            self.assertEqual([event['event'] for event in positive],
                             ['start', 'result', 'release', 'settled', 'applied'])
            self.assertEqual(len({event['id'] for event in positive}), 1)
            self.assertEqual(positive[1]['code'], 0)
            self.assertEqual(positive[0]['timeout'], 2500)
            self.assertLessEqual(positive[0]['at_ns'], positive[1]['completed_ns'])
            self.assertLessEqual(positive[1]['completed_ns'], positive[2]['at_ns'])
            events = evidence['fresh']['events']
            wanted = next(event for event in events if event['event'] == 'request-begin')
            fresh = next(event for event in events if event['event'] == 'completion'
                         and event.get('request_id') == wanted['request_id'])
            self.assertEqual(wanted['identity']['branch'], 'review/b')
            self.assertEqual(fresh['identity']['branch'], wanted['identity']['branch'])
            self.assertEqual(fresh['identity']['head'], wanted['identity']['head'])
            self.assertGreater(fresh['seq'], wanted['seq'])
            self.assertTrue(all(event['ok'] for event in events if event['event'] == 'request-end'))

    def test_arm_boundary_keeps_an_observation_started_before_rpc_returns(self):
        with tempfile.TemporaryDirectory(prefix='pair-observation-arm-') as storage:
            init = Path(storage) / 'init.lua'
            observed = Path(storage) / 'started-during-arm'
            init.write_text('local native=dofile\ndofile=function(path)\n'
                            ' local module=native(path)\n'
                            ' if path:match("/tests/lib/review_observation_gate%.lua$") then\n'
                            '  local install=module.install\n'
                            '  module.install=function(...)\n'
                            '   local gate=install(...); local arm=gate.arm\n'
                            '   gate.arm=function(self)\n'
                            '    local ok=arm(self)\n'
                            '    if ok then vim.api.nvim_exec_autocmds("FocusGained",{}); local started=vim.wait(2000,function()\n'
                            '     return gate:snapshot().inflight>0 or gate:snapshot().held>0\n'
                            '    end,5); if started then vim.fn.writefile({"started"},' + json.dumps(str(observed)) + ') end end\n'
                            '    return ok\n'
                            '   end\n'
                            '   return gate\n'
                            '  end\n'
                            ' end\n'
                            ' return module\n'
                            'end\n'
                            'native(' + json.dumps(str(ROOT / 'nvim/review.lua')) + ')\n')
            result = subprocess.run(['bash', str(ROOT / 'tests/review-observation-test.sh')],
                                    env=dict(os.environ, PAIR_TEST_REVIEW_INIT=str(init)),
                                    text=True, capture_output=True, timeout=60)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            self.assertTrue(observed.exists(), 'controlled observation never started during arm')

    def test_callback_gate_preserves_native_deadline_and_owned_teardown(self):
        with ReviewTestEnvironment() as isolated, tempfile.TemporaryDirectory(prefix='pair-observation-gate-') as storage:
            result = subprocess.run(['nvim', '--headless', '-u', 'NONE', '-l', str(ROOT / 'tests/review-observation-gate-test.lua')],
                                    cwd=ROOT, env=isolated.environment(storage, ROOT),
                                    text=True, capture_output=True, timeout=20)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_lua_trace_preserves_real_process_and_controlled_completion_seams(self):
        with ReviewTestEnvironment() as isolated, tempfile.TemporaryDirectory(prefix='pair-observation-trace-') as storage:
            result = subprocess.run(['nvim', '-l', str(ROOT / 'tests/review-observation-trace-test.lua')],
                                    cwd=ROOT, env=isolated.environment(storage, ROOT),
                                    text=True, capture_output=True, timeout=20)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_invalid_wanted_stops_before_launching_the_pane(self):
        with tempfile.TemporaryDirectory(prefix='pair-wanted-refusal-') as storage:
            temp = Path(storage)
            launched = temp / 'launched'
            init = temp / 'init.lua'
            init.write_text('vim.fn.writefile({"launched"},' + json.dumps(str(launched)) + ')\n')
            binary = temp / 'pair'
            binary.write_text('#!' + sys.executable + '\nimport json,sys\n'
                              'print(json.dumps(dict(status="invalid",repo=sys.argv[-1],'
                              'diagnostic="fixture wanted deadline")))\n')
            binary.chmod(0o700)
            result = subprocess.run(['bash', str(ROOT / 'tests/review-observation-test.sh')],
                                    env=dict(os.environ, PAIR_TEST_OBSERVATION_WANTED_BIN=str(binary),
                                             PAIR_TEST_REVIEW_INIT=str(init)),
                                    text=True, capture_output=True, timeout=20)
            self.assertNotEqual(result.returncode, 0)
            self.assertFalse(launched.exists(), 'invalid wanted observation reached the pane')
            for required in ('before forwarding', 'wanted:startup', 'fixture wanted deadline',
                             '"returncode": 0', '"status": "invalid"'):
                self.assertIn(required, result.stdout + result.stderr)

    def test_incomplete_fresh_and_real_checkout_change_keep_both_observations(self):
        for cause in ('invalid-fresh', 'changed-branch'):
            with self.subTest(cause=cause), tempfile.TemporaryDirectory(prefix='pair-observation-refusal-') as storage:
                init = Path(storage) / 'init.lua'
                setup = '''
local fault=false
local system=vim.system
vim.system=function(cmd,opts,callback)
  local process=system(cmd,opts,callback)
  if fault and cmd[2]=='review' and cmd[3]=='readiness' and cmd[4]=='--resolve' and not callback then
    local wait=process.wait
    process.wait=function(self,...)
      local result=wait(self,...)
      result.stdout=vim.json.encode({status='invalid',repo=cmd[5],diagnostic='fixture fresh deadline'})
      return result
    end
  end
  return process
end
'''
                action = 'fault=true'
                if cause == 'changed-branch':
                    setup = ''
                    action = '''
local request=type(raw)=='table' and raw or vim.json.decode(raw)
local result=vim.system({'git','-C',request.identity.repo,'checkout','-q','review/a'}):wait()
assert(result.code==0,result.stderr)
'''
                init.write_text(setup + 'dofile(' + json.dumps(str(ROOT / 'nvim/review.lua')) + ')\n' + '''
local restore=PairReviewPane.restore
PairReviewPane.restore=function(raw)
  local context=vim.deepcopy(PairReviewPane.controller:context())
  local buf=vim.api.nvim_get_current_buf()
  local lines=vim.api.nvim_buf_get_lines(buf,0,-1,false)
''' + action + '''
  local ack=restore(raw)
  assert(not ack.ok,'controlled observation unexpectedly activated')
  assert(vim.deep_equal(context,PairReviewPane.controller:context()),'refusal changed activation')
  assert(buf==vim.api.nvim_get_current_buf(),'refusal changed buffer')
  assert(vim.deep_equal(lines,vim.api.nvim_buf_get_lines(buf,0,-1,false)),'refusal changed bytes')
  return ack
end
''')
                result = subprocess.run(['bash', str(ROOT / 'tests/review-observation-test.sh')],
                                        env=dict(os.environ, PAIR_TEST_REVIEW_INIT=str(init)),
                                        text=True, capture_output=True, timeout=20)
                self.assertNotEqual(result.returncode, 0)
                prefix = 'review observation failure: '
                reports = [line[len(prefix):] for line in result.stderr.splitlines() if line.startswith(prefix)]
                self.assertTrue(reports, result.stdout + result.stderr)
                evidence = json.loads(reports[-1])
                self.assertEqual(evidence['phase'], 'restore:b')
                self.assertEqual(evidence['wanted'][-1]['identity']['branch'], 'review/b')
                events = evidence['fresh']['events']
                wanted = next(event for event in events if event['event'] == 'request-begin')
                self.assertEqual(wanted['identity']['branch'], 'review/b')
                fresh = [event for event in events if event['event'] == 'completion'
                         and event.get('request_id') == wanted['request_id']]
                self.assertTrue(fresh, evidence)
                self.assertGreater(fresh[0]['seq'], wanted['seq'])
                if cause == 'invalid-fresh':
                    self.assertEqual(fresh[0]['identity']['status'], 'invalid')
                    self.assertIn('fixture fresh deadline', fresh[0]['identity']['diagnostic'])
                else:
                    self.assertEqual(fresh[0]['identity']['status'], 'resolved')
                    self.assertEqual(fresh[0]['identity']['branch'], 'review/a')
                ending = next(event for event in events if event['event'] == 'request-end')
                self.assertFalse(ending['ok'])


if __name__ == '__main__':
    unittest.main()
