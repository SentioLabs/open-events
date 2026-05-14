from __future__ import annotations

from typing import Any

from .event_names import device, user
from .schemas import device as device_schemas
from .schemas import user as user_schemas

DISPATCH: dict[str, Any] = {
    user.AUTH_SIGNUP_V1: user_schemas.AUTH_SIGNUP_SCHEMA,
    user.AUTH_LOGIN_V1: user_schemas.AUTH_LOGIN_SCHEMA,
    user.AUTH_LOGOUT_V1: user_schemas.AUTH_LOGOUT_SCHEMA,
    user.CART_CHECKOUT_V1: user_schemas.CART_CHECKOUT_SCHEMA,
    user.CART_PURCHASE_V1: user_schemas.CART_PURCHASE_SCHEMA,
    user.CART_ITEM_ADDED_V1: user_schemas.CART_ITEM_ADDED_SCHEMA,
    device.INFO_HARDWARE_V1: device_schemas.INFO_HARDWARE_SCHEMA,
    device.INFO_SOFTWARE_V1: device_schemas.INFO_SOFTWARE_SCHEMA,
    device.INFO_CALIBRATION_V1: device_schemas.INFO_CALIBRATION_SCHEMA,
    device.INCIDENT_TEMPERATURE_V1: device_schemas.INCIDENT_TEMPERATURE_SCHEMA,
    device.INCIDENT_DROP_V1: device_schemas.INCIDENT_DROP_SCHEMA,
    device.DIAGNOSTICS_STACK_USAGE_V1: device_schemas.DIAGNOSTICS_STACK_USAGE_SCHEMA,
}


def decode(event_name: str, body: dict[str, Any]) -> dict[str, Any]:
    """Validate that body is for a known event, then return it unchanged.

    The API now sends JSON-encoded flat dicts over SQS (not base64 proto).
    This function validates the event_name is known and returns the row dict.
    Raises ValueError for unknown event names.
    """
    if event_name not in DISPATCH:
        raise ValueError(f"unknown event_name: {event_name!r}")
    return body
