from __future__ import annotations

import base64

import polars as pl
import pytest
from google.protobuf.json_format import MessageToDict

from consumer.dispatch import DISPATCH, decode, schema_for
from consumer.event_names import device, user

ALL_EVENT_NAMES = [
    user.AUTH_SIGNUP_V1,
    user.AUTH_LOGIN_V1,
    user.AUTH_LOGOUT_V1,
    user.CART_CHECKOUT_V1,
    user.CART_PURCHASE_V1,
    user.CART_ITEM_ADDED_V1,
    device.INFO_HARDWARE_V1,
    device.INFO_SOFTWARE_V1,
    device.INFO_CALIBRATION_V1,
    device.INCIDENT_TEMPERATURE_V1,
    device.INCIDENT_DROP_V1,
    device.DIAGNOSTICS_STACK_USAGE_V1,
]


def test_dispatch_has_all_12_events():
    assert len(DISPATCH) == 12


@pytest.mark.parametrize("name", ALL_EVENT_NAMES)
def test_dispatch_maps_each_event_to_tuple(name):
    assert name in DISPATCH, f"{name!r} not in DISPATCH"
    entry = DISPATCH[name]
    assert isinstance(entry, tuple) and len(entry) == 2, (
        f"DISPATCH[{name!r}] must be a (proto_class, schema) tuple"
    )
    proto_cls, schema = entry
    assert isinstance(schema, pl.Schema), (
        f"DISPATCH[{name!r}][1] is not pl.Schema, got {type(schema)}"
    )


@pytest.mark.parametrize("name", ALL_EVENT_NAMES)
def test_schema_for_returns_polars_schema(name):
    schema = schema_for(name)
    assert isinstance(schema, pl.Schema), (
        f"schema_for({name!r}) is not pl.Schema, got {type(schema)}"
    )


def test_dispatch_keys_match_event_name_constants():
    assert set(DISPATCH.keys()) == set(ALL_EVENT_NAMES)


def test_dispatch_info_hardware_schema_has_nested_struct():
    schema = schema_for(device.INFO_HARDWARE_V1)
    assert isinstance(schema["eeprom_format_version"], pl.Struct)


def test_dispatch_diagnostics_stack_usage_schema_has_list_of_struct():
    schema = schema_for(device.DIAGNOSTICS_STACK_USAGE_V1)
    assert isinstance(schema["threads"], pl.List)


def test_decode_user_auth_signup_roundtrip():
    """Encode a real proto message and verify decode() returns the expected fields."""
    from com.acme.platform.user.v1 import events_pb2 as user_pb2
    from google.protobuf import timestamp_pb2

    msg = user_pb2.UserAuthSignupV1()
    msg.event_name = user.AUTH_SIGNUP_V1
    msg.event_version = 1
    msg.event_id = "test-uuid-1234"
    ts = timestamp_pb2.Timestamp()
    ts.seconds = 1700000000
    msg.event_ts.CopyFrom(ts)
    msg.properties.method = user_pb2.UserAuthSignupV1Properties.METHOD_EMAIL
    msg.properties.plan = "starter"
    msg.context.tenant_id = "acme"
    msg.context.user_id = "u-1"
    msg.context.session_id = "s-1"
    msg.context.platform = user_pb2.UserContext.PLATFORM_WEB

    wire = msg.SerializeToString()
    body_b64 = base64.b64encode(wire).decode()

    result = decode(user.AUTH_SIGNUP_V1, body_b64)

    assert result["event_name"] == user.AUTH_SIGNUP_V1
    assert result["event_id"] == "test-uuid-1234"
    assert result["tenant_id"] == "acme"
    assert result["user_id"] == "u-1"
    assert result["plan"] == "starter"
    # method comes out as the enum name string via MessageToDict
    assert "method" in result
