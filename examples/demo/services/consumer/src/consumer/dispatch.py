from __future__ import annotations

import base64
import re
from datetime import datetime
from typing import Any

from com.acme.platform.device.v1 import events_pb2 as device_events_pb2
from com.acme.platform.user.v1 import events_pb2 as user_events_pb2
from google.protobuf.json_format import MessageToDict

# RFC3339-with-Z; matches what google.protobuf.Timestamp serializes to via MessageToDict.
_ISO_TS = re.compile(r"^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z$")


def _coerce_timestamps(obj: Any) -> Any:
    """Recursively convert RFC3339-Z strings into timezone-aware datetimes.

    google.protobuf.Timestamp values come out of MessageToDict as
    RFC3339 strings; Polars Datetime columns reject strings.
    """
    if isinstance(obj, dict):
        return {k: _coerce_timestamps(v) for k, v in obj.items()}
    if isinstance(obj, list):
        return [_coerce_timestamps(v) for v in obj]
    if isinstance(obj, str) and _ISO_TS.match(obj):
        return datetime.fromisoformat(obj.replace("Z", "+00:00"))
    return obj

from .event_names import device, user
from .schemas import device as device_schemas
from .schemas import user as user_schemas

# Map each canonical event-name@version to (proto-message-class, polars-schema).
DISPATCH: dict[str, tuple[type, Any]] = {
    user.AUTH_SIGNUP_V1:    (user_events_pb2.UserAuthSignupV1,    user_schemas.AUTH_SIGNUP_SCHEMA),
    user.AUTH_LOGIN_V1:     (user_events_pb2.UserAuthLoginV1,     user_schemas.AUTH_LOGIN_SCHEMA),
    user.AUTH_LOGOUT_V1:    (user_events_pb2.UserAuthLogoutV1,    user_schemas.AUTH_LOGOUT_SCHEMA),
    user.CART_CHECKOUT_V1:  (user_events_pb2.UserCartCheckoutV1,  user_schemas.CART_CHECKOUT_SCHEMA),
    user.CART_PURCHASE_V1:  (user_events_pb2.UserCartPurchaseV1,  user_schemas.CART_PURCHASE_SCHEMA),
    user.CART_ITEM_ADDED_V1: (user_events_pb2.UserCartItemAddedV1, user_schemas.CART_ITEM_ADDED_SCHEMA),
    device.INFO_HARDWARE_V1:    (device_events_pb2.DeviceInfoHardwareV1,    device_schemas.INFO_HARDWARE_SCHEMA),
    device.INFO_SOFTWARE_V1:    (device_events_pb2.DeviceInfoSoftwareV1,    device_schemas.INFO_SOFTWARE_SCHEMA),
    device.INFO_CALIBRATION_V1: (device_events_pb2.DeviceInfoCalibrationV1, device_schemas.INFO_CALIBRATION_SCHEMA),
    device.INCIDENT_TEMPERATURE_V1: (device_events_pb2.DeviceIncidentTemperatureV1, device_schemas.INCIDENT_TEMPERATURE_SCHEMA),
    device.INCIDENT_DROP_V1:        (device_events_pb2.DeviceIncidentDropV1,        device_schemas.INCIDENT_DROP_SCHEMA),
    device.DIAGNOSTICS_STACK_USAGE_V1: (device_events_pb2.DeviceDiagnosticsStackUsageV1, device_schemas.DIAGNOSTICS_STACK_USAGE_SCHEMA),
}


def schema_for(event_name: str) -> Any:
    """Return the Polars schema for an event name. Raises KeyError if unknown."""
    return DISPATCH[event_name][1]


def decode(event_name: str, body_b64: str) -> dict[str, Any]:
    """Base64-decode + proto-unmarshal an SQS message body into a flat dict.

    The API publishes events as base64(proto.Marshal(msg)) with the event_name
    in an SQS message attribute. This function looks up the proto class for
    event_name, parses the wire bytes, and returns a dict whose keys are the
    proto field names (preserved as snake_case via preserving_proto_field_name).
    """
    entry = DISPATCH.get(event_name)
    if entry is None:
        raise ValueError(f"unknown event_name: {event_name!r}")
    proto_cls, _schema = entry
    wire = base64.b64decode(body_b64)
    msg = proto_cls()
    msg.ParseFromString(wire)
    return _flatten_envelope(_coerce_timestamps(MessageToDict(msg, preserving_proto_field_name=True)))


def _flatten_envelope(d: dict[str, Any]) -> dict[str, Any]:
    """Hoist `context` and `properties` sub-fields to top-level.

    Proto wire format nests context fields under a `context` message and event
    fields under a `properties` message. Polars schemas express both as flat
    columns alongside envelope fields, so we hoist them here.
    """
    out: dict[str, Any] = {}
    for k, v in d.items():
        if k in ("context", "properties") and isinstance(v, dict):
            out.update(v)
        else:
            out[k] = v
    return out
