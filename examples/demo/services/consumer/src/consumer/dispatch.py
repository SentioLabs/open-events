from __future__ import annotations

import base64
from typing import Any

from google.protobuf.json_format import MessageToDict

from com.acme.storefront.v1 import events_pb2

from .event_names import (
    CHECKOUT_COMPLETED_V1,
    CHECKOUT_STARTED_V1,
    SEARCH_PERFORMED_V1,
)


EVENT_REGISTRY: dict[str, type] = {
    CHECKOUT_STARTED_V1:   events_pb2.CheckoutStartedV1,
    CHECKOUT_COMPLETED_V1: events_pb2.CheckoutCompletedV1,
    SEARCH_PERFORMED_V1:   events_pb2.SearchPerformedV1,
}


def decode(event_name: str, body_b64: str) -> dict[str, Any]:
    cls = EVENT_REGISTRY.get(event_name)
    if cls is None:
        raise ValueError(f"unknown event_name: {event_name!r}")
    wire = base64.b64decode(body_b64)
    msg = cls()
    msg.ParseFromString(wire)
    return MessageToDict(msg, preserving_proto_field_name=True)
