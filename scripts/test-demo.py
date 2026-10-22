import importlib.util
from pathlib import Path
import sys
import unittest

spec = importlib.util.spec_from_file_location("demo", Path(__file__).with_name("demo.py"))
demo = importlib.util.module_from_spec(spec)
spec.loader.exec_module(demo)


class DemoTests(unittest.TestCase):
    def test_prompt_reply(self):
        output = demo.capture([sys.executable, "-c",
                               "print('Name?', flush=True); print('Hello', input())"],
                              {}, [("Name?", "Alex")])
        self.assertIn("Hello Alex", output)

    def test_failure_is_not_hidden(self):
        with self.assertRaises(RuntimeError):
            demo.capture([sys.executable, "-c", "raise SystemExit(2)"], {})

    def test_missing_prompt(self):
        with self.assertRaises(RuntimeError):
            demo.capture([sys.executable, "-c", "print('done')"], {}, [("Name?", "Alex")])

    def test_timeout(self):
        with self.assertRaises(TimeoutError):
            demo.capture([sys.executable, "-c", "import time; time.sleep(5)"], {}, timeout=0.2)

    def test_briefing_contract(self):
        demo.validate_briefing("Alex UTC\nTIP | Docker")
        for text in ["Alex UTC", "Alex UTC\nTIP | Docker\nTIP | AWS",
                     "Someone UTC\nTIP | AWS", "Alex UTC\x1b[31m\nTIP | AWS"]:
            with self.assertRaises(ValueError):
                demo.validate_briefing(text)


if __name__ == "__main__":
    unittest.main()
