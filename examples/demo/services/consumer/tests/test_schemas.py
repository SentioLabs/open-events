from __future__ import annotations

import polars as pl

from consumer.event_names import (
    ALL_EVENT_NAMES,
    CHECKOUT_STARTED_V1,
)
from consumer.schemas import EVENT_SCHEMAS


def test_every_event_has_a_schema():
    for name in ALL_EVENT_NAMES:
        assert name in EVENT_SCHEMAS
        assert isinstance(EVENT_SCHEMAS[name], pl.Schema)


def test_checkout_started_schema_has_expected_fields():
    sch = EVENT_SCHEMAS[CHECKOUT_STARTED_V1]
    # envelope
    assert sch["event_id"] == pl.Utf8
    assert sch["event_version"] == pl.Int64
    # nested context is a struct
    assert isinstance(sch["context"], pl.Struct)
    # nested properties
    assert isinstance(sch["properties"], pl.Struct)
