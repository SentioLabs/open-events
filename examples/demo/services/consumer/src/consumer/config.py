from __future__ import annotations

import os
from dataclasses import dataclass


@dataclass(frozen=True)
class Settings:
    queue_url: str
    endpoint_url: str | None
    region: str
    output_dir: str
    batch_size: int
    flush_interval_s: float

    @classmethod
    def from_env(cls) -> "Settings":
        return cls(
            queue_url=os.environ["OPENEVENTS_QUEUE_URL"],
            endpoint_url=os.environ.get("AWS_ENDPOINT_URL"),
            region=os.environ.get("AWS_REGION", "us-east-1"),
            output_dir=os.environ.get("OPENEVENTS_OUTPUT_DIR", "../../../../_build/demo-output"),
            batch_size=int(os.environ.get("OPENEVENTS_BATCH_SIZE", "10")),
            flush_interval_s=float(os.environ.get("OPENEVENTS_FLUSH_INTERVAL_S", "5")),
        )
