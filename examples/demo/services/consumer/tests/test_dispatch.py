from __future__ import annotations

import polars as pl
import pytest

from consumer.dispatch import DISPATCH
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
def test_dispatch_maps_each_event_to_schema(name):
    assert name in DISPATCH, f"{name!r} not in DISPATCH"
    assert isinstance(DISPATCH[name], pl.Schema), f"DISPATCH[{name!r}] is not pl.Schema"


def test_dispatch_keys_match_event_name_constants():
    assert set(DISPATCH.keys()) == set(ALL_EVENT_NAMES)


def test_dispatch_info_hardware_schema_has_nested_struct():
    schema = DISPATCH[device.INFO_HARDWARE_V1]
    assert isinstance(schema["eeprom_format_version"], pl.Struct)


def test_dispatch_diagnostics_stack_usage_schema_has_list_of_struct():
    schema = DISPATCH[device.DIAGNOSTICS_STACK_USAGE_V1]
    assert isinstance(schema["threads"], pl.List)
