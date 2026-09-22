import json
import logging
import traceback
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen

logger = logging.getLogger(__name__)


class Faultwing:
    def __init__(
        self,
        url: str,
        api_key: str,
        *,
        environment: str = "production",
        release: str | None = None,
        timeout: float = 2.0,
    ) -> None:
        if not url.strip():
            raise ValueError("url is required")
        if not api_key.strip():
            raise ValueError("api_key is required")
        if not environment.strip():
            raise ValueError("environment is required")
        if timeout <= 0:
            raise ValueError("timeout must be greater than zero")

        base_url = url.strip().rstrip("/")
        release_name = release.strip() if release else ""

        self._events_url = f"{base_url}/api/v1/events"
        self._api_key = api_key.strip()
        self._environment = environment.strip()
        self._release = release_name or None
        self._timeout = timeout

    def capture_exception(self, exception: Exception) -> bool:
        """Send an exception to Faultwing without raising delivery errors."""
        exception_type = type(exception).__name__
        payload = {
            "exception_type": exception_type,
            "message": str(exception) or exception_type,
            "stacktrace": "".join(
                traceback.format_exception(
                    type(exception),
                    exception,
                    exception.__traceback__,
                )
            ),
            "environment": self._environment,
        }
        if self._release is not None:
            payload["release"] = self._release

        request = Request(
            self._events_url,
            data=json.dumps(payload).encode("utf-8"),
            headers={
                "Authorization": f"Bearer {self._api_key}",
                "Content-Type": "application/json",
                "User-Agent": "faultwing-python/0.1.0",
            },
            method="POST",
        )

        try:
            with urlopen(request, timeout=self._timeout) as response:
                return 200 <= response.status < 300
        except (HTTPError, URLError, OSError, TimeoutError) as error:
            logger.warning("could not send exception to Faultwing: %s", error)
            return False
