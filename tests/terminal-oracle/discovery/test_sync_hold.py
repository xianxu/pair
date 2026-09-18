import unittest

import sync_hold


class VerdictTest(unittest.TestCase):
    def test_honoured_needs_the_marker_after_the_close(self):
        control = dict(marker_after=0.01)
        self.assertEqual(sync_hold.verdict(control, dict(marker_after=1.02, closed_after=1.0)), 0)
        self.assertEqual(sync_hold.verdict(control, dict(marker_after=0.01, closed_after=1.0)), 1)

    def test_a_marker_that_never_arrives_is_not_honoured(self):
        # A held pane that never renders at all is a failure, not a pass.
        self.assertEqual(sync_hold.verdict(dict(marker_after=0.01), dict(marker_after=None, closed_after=1.0)), 1)

    def test_a_control_that_measured_nothing_is_inconclusive(self):
        held = dict(marker_after=1.02, closed_after=1.0)
        self.assertEqual(sync_hold.verdict(dict(marker_after=None), held), 2)
        self.assertEqual(sync_hold.verdict(dict(marker_after=0.9), held), 2)


class InstrumentFailureTest(unittest.TestCase):
    def test_a_crash_is_inconclusive_not_a_verdict(self):
        def broken(bracket):
            raise OSError('out of pty devices')
        original = sync_hold.run
        sync_hold.run = broken
        try:
            self.assertEqual(sync_hold.main(), 2)
        finally:
            sync_hold.run = original


if __name__ == '__main__':
    unittest.main()
