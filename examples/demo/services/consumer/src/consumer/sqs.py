from __future__ import annotations

import logging
import threading
from typing import Any

from .attrs import ATTR_EVENT_NAME
from .dispatch import decode
from .sink import Sink

log = logging.getLogger("consumer.sqs")


def poll(
    client: Any,
    queue_url: str,
    sink: Sink,
    stop_event: threading.Event,
    *,
    until_empty: bool = False,
    wait_time_s: int = 20,
    max_messages: int = 10,
) -> None:
    empty_streak = 0
    while not stop_event.is_set():
        resp = client.receive_message(
            QueueUrl=queue_url,
            WaitTimeSeconds=wait_time_s,
            MaxNumberOfMessages=max_messages,
            MessageAttributeNames=["All"],
        )
        messages = resp.get("Messages", [])
        if not messages:
            empty_streak += 1
            if until_empty and empty_streak >= 2:
                break
            sink.maybe_flush()
            continue
        empty_streak = 0
        for msg in messages:
            attrs = msg.get("MessageAttributes", {})
            name = attrs.get(ATTR_EVENT_NAME, {}).get("StringValue")
            if name is None:
                log.warning("skipping message %s: missing %s", msg["MessageId"], ATTR_EVENT_NAME)
                continue
            try:
                row = decode(name, msg["Body"])
            except Exception as exc:
                log.error("decode failed for %s: %s", msg["MessageId"], exc)
                continue
            sink.append(name, row)
            client.delete_message(QueueUrl=queue_url, ReceiptHandle=msg["ReceiptHandle"])
        sink.maybe_flush()
    sink.flush_all()
