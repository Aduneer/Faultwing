import json
import unittest
from unittest.mock import patch
from urllib.error import URLError

from flytrap import FlyTrap


class FakeResponse:
    status = 202

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        return False


class FlyTrapTest(unittest.TestCase):
    @patch("flytrap.client.urlopen", return_value=FakeResponse())
    def test_capture_exception_sends_monitoring_context(self, mocked_urlopen):
        client = FlyTrap(
            "http://localhost:8080/",
            "fly_test-key",
            environment="development",
            release="1.3.2",
            timeout=1.5,
        )

        try:
            raise ValueError("something broke")
        except ValueError as error:
            delivered = client.capture_exception(error)

        self.assertTrue(delivered)
        request = mocked_urlopen.call_args.args[0]
        self.assertEqual(request.full_url, "http://localhost:8080/api/v1/events")
        self.assertEqual(request.get_header("Authorization"), "Bearer fly_test-key")
        self.assertEqual(mocked_urlopen.call_args.kwargs["timeout"], 1.5)

        payload = json.loads(request.data)
        self.assertEqual(payload["exception_type"], "ValueError")
        self.assertEqual(payload["message"], "something broke")
        self.assertIn("raise ValueError", payload["stacktrace"])
        self.assertEqual(payload["environment"], "development")
        self.assertEqual(payload["release"], "1.3.2")

    @patch("flytrap.client.urlopen", side_effect=URLError("offline"))
    def test_capture_exception_does_not_raise_delivery_errors(self, _mocked_urlopen):
        client = FlyTrap("http://localhost:8080", "fly_test-key")

        with self.assertLogs("flytrap.client", level="WARNING"):
            delivered = client.capture_exception(RuntimeError("application error"))

        self.assertFalse(delivered)


if __name__ == "__main__":
    unittest.main()
