from __future__ import annotations

import datetime as dt
import pathlib
import time
from typing import Any

import polars as pl

from .schemas import EVENT_SCHEMAS


def _safe_dir(name: str) -> str:
    # "checkout.started@1" -> "checkout_started_v1"
    n = name.replace("@", "_v").replace(".", "_")
    return n


class Sink:
    def __init__(self, output_dir: str | pathlib.Path, batch_size: int = 10, flush_interval_s: float = 5.0):
        self.output_dir = pathlib.Path(output_dir)
        self.batch_size = batch_size
        self.flush_interval_s = flush_interval_s
        self._buffers: dict[str, list[dict[str, Any]]] = {}
        self._last_flush = time.monotonic()

    def append(self, event_name: str, row: dict[str, Any]) -> None:
        self._buffers.setdefault(event_name, []).append(row)

    def maybe_flush(self) -> None:
        now = time.monotonic()
        full = any(len(rows) >= self.batch_size for rows in self._buffers.values())
        if full or (now - self._last_flush) >= self.flush_interval_s:
            self.flush_all()
            self._last_flush = now

    def flush_all(self) -> None:
        for name, rows in list(self._buffers.items()):
            if not rows:
                continue
            self._write(name, rows)
            self._buffers[name] = []

    def _write(self, event_name: str, rows: list[dict[str, Any]]) -> None:
        schema = EVENT_SCHEMAS[event_name]
        df = pl.DataFrame(rows, schema=schema)
        ts = dt.datetime.now(dt.UTC).strftime("%Y%m%dT%H%M%SZ")
        out = self.output_dir / _safe_dir(event_name)
        out.mkdir(parents=True, exist_ok=True)
        df.write_parquet(out / f"{ts}-{id(rows):x}.parquet")
