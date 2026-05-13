from __future__ import annotations

import pathlib

import polars as pl

from consumer.event_names import CHECKOUT_STARTED_V1
from consumer.sink import Sink


def _row():
    return {
        "event_name": "checkout.started",
        "event_version": 1,
        "event_id": "abc",
        "event_ts": "2026-01-01T00:00:00Z",
        "client": {"name": "demo-api", "version": "0.1.0"},
        "context": {"tenant_id": "acme", "user_id": None, "session_id": None, "platform": "PLATFORM_WEB"},
        "properties": {"cart_id": "cart-1", "currency": "CURRENCY_USD", "item_count": 1, "subtotal_cents": 100},
    }


def test_flush_writes_parquet(tmp_path: pathlib.Path):
    sink = Sink(tmp_path, batch_size=2, flush_interval_s=999)
    sink.append(CHECKOUT_STARTED_V1, _row())
    sink.append(CHECKOUT_STARTED_V1, _row())
    sink.maybe_flush()

    parquets = list(tmp_path.rglob("*.parquet"))
    assert len(parquets) == 1
    df = pl.read_parquet(parquets[0])
    assert df.height == 2
    assert "event_id" in df.columns
