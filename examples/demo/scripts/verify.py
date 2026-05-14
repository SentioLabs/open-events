"""Summarize Parquet output written by the demo consumer.

Reads each event-type subdirectory under the supplied root and prints a short
header plus the dataframe contents for each. The directory layout matches
`consumer.sink.safe_dir` — a per-event-type directory holding one parquet file
per flushed batch.
"""

from __future__ import annotations

import pathlib
import sys

import polars as pl


def main(root: pathlib.Path) -> int:
    if not root.exists():
        print(f"no parquet output under {root}")
        return 0
    found = False
    for event_dir in sorted(p for p in root.iterdir() if p.is_dir()):
        files = sorted(event_dir.glob("*.parquet"))
        if not files:
            continue
        found = True
        df = pl.read_parquet(files)
        print(f"\n=== {event_dir.name} ({df.height} rows) ===")
        print(df)
    if not found:
        print(f"no parquet output under {root}")
    return 0


if __name__ == "__main__":
    arg = sys.argv[1] if len(sys.argv) > 1 else "_build/demo-output"
    sys.exit(main(pathlib.Path(arg)))
