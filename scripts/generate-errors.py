#!/usr/bin/env python3

import argparse
import os
import random
import time
from collections.abc import Callable, Sequence

from faultwing import Faultwing


class DatabaseTimeoutError(RuntimeError):
    pass


class InvalidOrderError(ValueError):
    pass


class UpstreamConnectionError(ConnectionError):
    pass


def query_database() -> None:
    raise DatabaseTimeoutError("database connection timed out after 5 seconds")


def validate_order() -> None:
    order_id = random.randint(1000, 9999)
    raise InvalidOrderError(f"order {order_id} has no line items")


def call_upstream_service() -> None:
    raise UpstreamConnectionError("payment service refused the connection")


ERROR_FACTORIES: dict[str, Callable[[], None]] = {
    "database": query_database,
    "validation": validate_order,
    "network": call_upstream_service,
}


def positive_integer(value: str) -> int:
    number = int(value)
    if number < 1:
        raise argparse.ArgumentTypeError("must be at least 1")
    return number


def non_negative_float(value: str) -> float:
    number = float(value)
    if number < 0:
        raise argparse.ArgumentTypeError("must be zero or greater")
    return number


def parse_args(argv: Sequence[str] | None = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Send fake exceptions to Faultwing")
    parser.add_argument(
        "--url",
        default=os.getenv("FAULTWING_URL", "http://localhost:8080"),
        help="Faultwing base URL (default: %(default)s)",
    )
    parser.add_argument(
        "--api-key",
        default=os.getenv("FAULTWING_API_KEY"),
        help="project API key (or set FAULTWING_API_KEY)",
    )
    parser.add_argument(
        "--environment",
        default=os.getenv("FAULTWING_ENVIRONMENT", "development"),
        help="event environment (default: %(default)s)",
    )
    parser.add_argument(
        "--release",
        default=os.getenv("FAULTWING_RELEASE"),
        help="optional release name (or set FAULTWING_RELEASE)",
    )
    parser.add_argument(
        "--type",
        dest="error_type",
        choices=[*ERROR_FACTORIES, "random"],
        default="random",
        help="error type to generate (default: %(default)s)",
    )
    parser.add_argument(
        "--count",
        type=positive_integer,
        default=10,
        help="number of errors to send (default: %(default)s)",
    )
    parser.add_argument(
        "--delay",
        type=non_negative_float,
        default=0.1,
        help="seconds between errors; use 0 for a burst (default: %(default)s)",
    )

    args = parser.parse_args(argv)
    if not args.api_key:
        parser.error("--api-key or FAULTWING_API_KEY is required")
    return args


def main(argv: Sequence[str] | None = None) -> int:
    args = parse_args(argv)
    client = Faultwing(
        args.url,
        args.api_key,
        environment=args.environment,
        release=args.release,
    )

    delivered = 0
    factories = list(ERROR_FACTORIES.values())

    for index in range(args.count):
        factory = (
            random.choice(factories)
            if args.error_type == "random"
            else ERROR_FACTORIES[args.error_type]
        )

        try:
            factory()
        except Exception as error:
            if client.capture_exception(error):
                delivered += 1

        if args.delay and index < args.count - 1:
            time.sleep(args.delay)

    failed = args.count - delivered
    print(f"generated={args.count} delivered={delivered} failed={failed}")
    return 0 if failed == 0 else 1


if __name__ == "__main__":
    raise SystemExit(main())
