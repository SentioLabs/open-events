from __future__ import annotations

import base64

from com.acme.storefront.v1 import events_pb2
from consumer.dispatch import decode
from consumer.event_names import CHECKOUT_STARTED_V1


def _sample_started_bytes() -> bytes:
    msg = events_pb2.CheckoutStartedV1()
    msg.event_name = "checkout.started"
    msg.event_version = 1
    msg.event_id = "00000000-0000-0000-0000-000000000001"
    msg.context.tenant_id = "acme"
    msg.context.platform = events_pb2.Context.PLATFORM_WEB
    msg.properties.cart_id = "cart-1"
    msg.properties.item_count = 3
    msg.properties.subtotal_cents = 4999
    msg.properties.currency = events_pb2.CheckoutStartedV1Properties.CURRENCY_USD
    return msg.SerializeToString()


def test_decode_known_bytes_yields_dict():
    body_b64 = base64.b64encode(_sample_started_bytes()).decode()
    row = decode(CHECKOUT_STARTED_V1, body_b64)
    assert row["event_id"] == "00000000-0000-0000-0000-000000000001"
    assert row["context"]["tenant_id"] == "acme"
    assert row["context"]["platform"] == "PLATFORM_WEB"
    assert row["properties"]["currency"] == "CURRENCY_USD"


def test_decode_unknown_event_name_raises():
    import pytest
    with pytest.raises(ValueError):
        decode("nope.unknown@1", "")
