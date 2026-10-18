import importlib.util
import json
from pathlib import Path
import unittest


class FixtureReplyTest(unittest.TestCase):
    def setUp(self):
        spec = importlib.util.spec_from_file_location(
            "fixture_reply", Path(__file__).with_name("zellij_fixture_replies.py"))
        self.module = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.module)

    def test_native_clients_and_absence_are_different(self):
        zero = self.module.render("clients", [])
        self.assertEqual(zero, dict(stdout="CLIENT_ID ZELLIJ_PANE_ID RUNNING_COMMAND\n", stderr="", code=0))
        absent = self.module.render("no-session", None)
        self.assertEqual(absent, dict(stdout="", stderr="There is no active session!\n", code=1))

    def test_panes_keep_semantics_with_complete_native_fields(self):
        raw = self.module.render("panes", [{"id": 4, "terminal_command": "pair term", "is_focused": True, "pane_x": 75}])
        pane = json.loads(raw["stdout"])[0]
        self.assertEqual(pane["id"], 4)
        self.assertEqual(pane["pane_x"], 75)
        self.assertTrue(pane["is_focused"])
        self.assertFalse(pane["is_fullscreen"])
        self.assertEqual(pane["tab_id"], 2)
        self.assertGreater(pane["pane_rows"], 0)

    def test_fault_is_explicit_and_never_normalized(self):
        raw = dict(stdout="CLIENT_ID\n", stderr="", code=0)
        self.assertEqual(self.module.render("fault", raw), raw)
        with self.assertRaises(ValueError):
            self.module.render("unknown", [])

    def test_validators_are_targeted_and_keep_exact_status(self):
        reply = self.module.render("name-invalid", {"name": "pair-probe", "limit": 108})
        self.assertEqual(reply["code"], 2)
        self.assertEqual(reply["stdout"], "")
        self.assertIn("'pair-probe' for '--session <SESSION>'", reply["stderr"])

    def test_false_visibility_keeps_native_exit_semantics(self):
        self.assertEqual(self.module.render("visibility", False),
                         dict(stdout="false\n", stderr="", code=1))
        self.assertEqual(self.module.render("visibility", True),
                         dict(stdout="true\n", stderr="", code=0))


if __name__ == "__main__":
    unittest.main()
