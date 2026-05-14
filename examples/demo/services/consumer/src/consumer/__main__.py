from __future__ import annotations

import argparse
import logging
import signal
import sys
import threading
from types import FrameType

import boto3

from .config import Settings
from .sink import Sink
from .sqs import poll


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(prog="demo-consumer")
    parser.add_argument(
        "--until-empty",
        action="store_true",
        help="exit cleanly after two consecutive empty receives",
    )
    args = parser.parse_args(argv)

    logging.basicConfig(level=logging.INFO, format="%(asctime)s %(name)s %(message)s")
    settings = Settings.from_env()

    client = boto3.client("sqs", region_name=settings.region, endpoint_url=settings.endpoint_url)
    sink = Sink(settings.output_dir, settings.batch_size, settings.flush_interval_s)
    stop = threading.Event()

    def _handle(_signum: int, _frame: FrameType | None) -> None:
        stop.set()

    signal.signal(signal.SIGINT, _handle)
    signal.signal(signal.SIGTERM, _handle)

    poll(client, settings.queue_url, sink, stop, until_empty=args.until_empty)
    return 0


if __name__ == "__main__":
    sys.exit(main())
