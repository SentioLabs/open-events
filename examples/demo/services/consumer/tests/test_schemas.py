from __future__ import annotations

import polars as pl
import pytest

from consumer.event_names import device, user
from consumer.schemas import device as device_schemas
from consumer.schemas import user as user_schemas

ALL_EVENTS = [
    (user.AUTH_SIGNUP_V1, user_schemas.AUTH_SIGNUP_SCHEMA),
    (user.AUTH_LOGIN_V1, user_schemas.AUTH_LOGIN_SCHEMA),
    (user.AUTH_LOGOUT_V1, user_schemas.AUTH_LOGOUT_SCHEMA),
    (user.CART_CHECKOUT_V1, user_schemas.CART_CHECKOUT_SCHEMA),
    (user.CART_PURCHASE_V1, user_schemas.CART_PURCHASE_SCHEMA),
    (user.CART_ITEM_ADDED_V1, user_schemas.CART_ITEM_ADDED_SCHEMA),
    (device.INFO_HARDWARE_V1, device_schemas.INFO_HARDWARE_SCHEMA),
    (device.INFO_SOFTWARE_V1, device_schemas.INFO_SOFTWARE_SCHEMA),
    (device.INFO_CALIBRATION_V1, device_schemas.INFO_CALIBRATION_SCHEMA),
    (device.INCIDENT_TEMPERATURE_V1, device_schemas.INCIDENT_TEMPERATURE_SCHEMA),
    (device.INCIDENT_DROP_V1, device_schemas.INCIDENT_DROP_SCHEMA),
    (device.DIAGNOSTICS_STACK_USAGE_V1, device_schemas.DIAGNOSTICS_STACK_USAGE_SCHEMA),
]


@pytest.mark.parametrize("name,schema", ALL_EVENTS)
def test_schema_is_polars_schema(name, schema):
    assert isinstance(schema, pl.Schema), f"{name} schema is not pl.Schema"


def test_all_schemas_have_envelope_fields():
    envelope_fields = {"event_name", "event_version", "event_id", "event_ts"}
    for name, schema in ALL_EVENTS:
        for field in envelope_fields:
            assert field in schema, f"{name} schema missing envelope field {field!r}"


def test_user_schemas_have_user_context_fields():
    user_context_fields = {"tenant_id", "user_id", "session_id", "platform"}
    user_event_schemas = [
        user_schemas.AUTH_SIGNUP_SCHEMA,
        user_schemas.AUTH_LOGIN_SCHEMA,
        user_schemas.AUTH_LOGOUT_SCHEMA,
        user_schemas.CART_CHECKOUT_SCHEMA,
        user_schemas.CART_PURCHASE_SCHEMA,
        user_schemas.CART_ITEM_ADDED_SCHEMA,
    ]
    for schema in user_event_schemas:
        for field in user_context_fields:
            assert field in schema, f"user schema missing context field {field!r}"


def test_device_schemas_have_device_context_fields():
    device_context_fields = {"tenant_id", "device_id", "serial_number", "firmware_version"}
    device_event_schemas = [
        device_schemas.INFO_HARDWARE_SCHEMA,
        device_schemas.INFO_SOFTWARE_SCHEMA,
        device_schemas.INFO_CALIBRATION_SCHEMA,
        device_schemas.INCIDENT_TEMPERATURE_SCHEMA,
        device_schemas.INCIDENT_DROP_SCHEMA,
        device_schemas.DIAGNOSTICS_STACK_USAGE_SCHEMA,
    ]
    for schema in device_event_schemas:
        for field in device_context_fields:
            assert field in schema, f"device schema missing context field {field!r}"


def test_info_hardware_schema_has_nested_struct_fields():
    schema = device_schemas.INFO_HARDWARE_SCHEMA
    assert isinstance(schema["eeprom_format_version"], pl.Struct)
    assert isinstance(schema["module_pcb_version"], pl.Struct)
    # verify nested struct fields
    eeprom_fields = {f.name for f in schema["eeprom_format_version"].fields}
    assert "major" in eeprom_fields
    assert "minor" in eeprom_fields


def test_diagnostics_stack_usage_schema_has_list_of_struct():
    schema = device_schemas.DIAGNOSTICS_STACK_USAGE_SCHEMA
    assert isinstance(schema["threads"], pl.List)
    inner = schema["threads"].inner
    assert isinstance(inner, pl.Struct)
    thread_fields = {f.name for f in inner.fields}
    assert "name" in thread_fields
    assert "stack_size_bytes" in thread_fields
    assert "stack_used_bytes" in thread_fields
    assert "usage_percent" in thread_fields
    assert "priority" in thread_fields
    assert "state" in thread_fields


def test_auth_login_schema_has_property_fields():
    schema = user_schemas.AUTH_LOGIN_SCHEMA
    assert "method" in schema
    assert "success" in schema


def test_cart_purchase_schema_has_property_fields():
    schema = user_schemas.CART_PURCHASE_SCHEMA
    assert "cart_id" in schema
    assert "order_id" in schema
    assert "total_cents" in schema
    assert "payment_method" in schema
    assert "coupon_code" in schema
