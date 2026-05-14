from __future__ import annotations

import polars as pl

# ---------------------------------------------------------------------------
# Shared envelope and context columns for all device-domain events.
# ---------------------------------------------------------------------------
_ENVELOPE = {
    "event_name": pl.Utf8,
    "event_version": pl.Int32,
    "event_id": pl.Utf8,
    "event_ts": pl.Datetime("us", "UTC"),
}

_DEVICE_CONTEXT = {
    "tenant_id": pl.Utf8,
    "device_id": pl.Utf8,
    "serial_number": pl.Utf8,
    "firmware_version": pl.Utf8,
}

# ---------------------------------------------------------------------------
# device.info.hardware@1
# ---------------------------------------------------------------------------
INFO_HARDWARE_SCHEMA = pl.Schema({
    **_ENVELOPE,
    **_DEVICE_CONTEXT,
    "unique_id": pl.Utf8,
    "manufacturing_timestamp": pl.Datetime("us", "UTC"),
    "eeprom_format_version": pl.Struct({"major": pl.Int64, "minor": pl.Int64}),
    "module_pcb_version": pl.Struct({"major": pl.Int64, "minor": pl.Int64}),
    "sensor_type": pl.Utf8,
    "fuel_cell_lot_number": pl.Utf8,
    "fuel_cell_vendor": pl.Utf8,
})

# ---------------------------------------------------------------------------
# device.info.software@1
# ---------------------------------------------------------------------------
INFO_SOFTWARE_SCHEMA = pl.Schema({
    **_ENVELOPE,
    **_DEVICE_CONTEXT,
    "unique_id": pl.Int64,
    "product_type": pl.Utf8,
    "pcba_hw_version": pl.Utf8,
    "pcba_hw_manufactured_timestamp_ms": pl.Int64,
    "versions": pl.Struct({
        "zephyr_kernel_version": pl.Utf8,
        "zephyr_kernel_git_sha": pl.Utf8,
        "app_semantic_version": pl.Utf8,
        "app_git_sha": pl.Utf8,
        "compile_time": pl.Utf8,
        "compiled_on_os": pl.Utf8,
        "compiled_by": pl.Utf8,
        "compiler_version": pl.Utf8,
        "hw_platform": pl.Utf8,
        "build_type": pl.Utf8,
        "bootloader_git_sha": pl.Utf8,
        "bootloader_build_timestamp": pl.Int64,
    }),
})

# ---------------------------------------------------------------------------
# device.info.calibration@1
# ---------------------------------------------------------------------------
INFO_CALIBRATION_SCHEMA = pl.Schema({
    **_ENVELOPE,
    **_DEVICE_CONTEXT,
    "concentration": pl.Float64,
    "integral": pl.Int64,
    "timestamp": pl.Datetime("us", "UTC"),
})

# ---------------------------------------------------------------------------
# device.incident.temperature@1
# ---------------------------------------------------------------------------
INCIDENT_TEMPERATURE_SCHEMA = pl.Schema({
    **_ENVELOPE,
    **_DEVICE_CONTEXT,
    "degrees_c": pl.Float64,
    "threshold_c": pl.Float64,
    "breach_type": pl.Utf8,
})

# ---------------------------------------------------------------------------
# device.incident.drop@1
# ---------------------------------------------------------------------------
INCIDENT_DROP_SCHEMA = pl.Schema({
    **_ENVELOPE,
    **_DEVICE_CONTEXT,
    "peak_acceleration_g": pl.Float64,
    "axis": pl.Utf8,
    "duration_ms": pl.Int64,
})

# ---------------------------------------------------------------------------
# device.diagnostics.stack_usage@1
# ---------------------------------------------------------------------------
DIAGNOSTICS_STACK_USAGE_SCHEMA = pl.Schema({
    **_ENVELOPE,
    **_DEVICE_CONTEXT,
    "thread_count": pl.Int64,
    "highest_usage_percent": pl.Int64,
    "highest_usage_thread": pl.Utf8,
    "threads": pl.List(pl.Struct({
        "name": pl.Utf8,
        "stack_size_bytes": pl.Int64,
        "stack_used_bytes": pl.Int64,
        "usage_percent": pl.Int64,
        "priority": pl.Int64,
        "state": pl.Utf8,
    })),
})
