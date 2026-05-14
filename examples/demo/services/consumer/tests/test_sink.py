from __future__ import annotations

import datetime as dt
import pathlib

import polars as pl

from consumer.event_names import device, user
from consumer.sink import Sink, safe_dir

_TS = dt.datetime(2026, 1, 1, 0, 0, 0, tzinfo=dt.UTC)


def _user_auth_login_row():
    return {
        "event_name": "user.auth.login",
        "event_version": 1,
        "event_id": "abc",
        "event_ts": _TS,
        "tenant_id": "acme",
        "user_id": None,
        "session_id": None,
        "platform": "ios",
        "method": "email",
        "success": True,
    }


def _device_info_hardware_row():
    return {
        "event_name": "device.info.hardware",
        "event_version": 1,
        "event_id": "def",
        "event_ts": _TS,
        "tenant_id": "acme",
        "device_id": "dev-1",
        "serial_number": "SN-001",
        "firmware_version": "1.0.0",
        "unique_id": "uid-001",
        "manufacturing_timestamp": _TS,
        "eeprom_format_version": {"major": 1, "minor": 0},
        "module_pcb_version": {"major": 2, "minor": 1},
        "sensor_type": "co",
        "fuel_cell_lot_number": None,
        "fuel_cell_vendor": None,
    }


def _device_diagnostics_stack_usage_row():
    return {
        "event_name": "device.diagnostics.stack_usage",
        "event_version": 1,
        "event_id": "ghi",
        "event_ts": _TS,
        "tenant_id": "acme",
        "device_id": "dev-1",
        "serial_number": "SN-001",
        "firmware_version": "1.0.0",
        "thread_count": 2,
        "highest_usage_percent": 80,
        "highest_usage_thread": "main",
        "threads": [
            {
                "name": "main",
                "stack_size_bytes": 4096,
                "stack_used_bytes": 3276,
                "usage_percent": 80,
                "priority": 0,
                "state": "running",
            }
        ],
    }


def test_safe_dir_two_segment_name():
    assert safe_dir("user.auth.login@1") == "user_auth_login_v1"


def test_safe_dir_three_segment_name():
    assert safe_dir("device.diagnostics.stack_usage@1") == "device_diagnostics_stack_usage_v1"


def test_safe_dir_two_dot_segment():
    assert safe_dir("device.info.hardware@1") == "device_info_hardware_v1"


def test_flush_writes_parquet_for_user_event(tmp_path: pathlib.Path):
    sink = Sink(tmp_path, batch_size=2, flush_interval_s=999)
    sink.append(user.AUTH_LOGIN_V1, _user_auth_login_row())
    sink.append(user.AUTH_LOGIN_V1, _user_auth_login_row())
    sink.maybe_flush()

    parquets = list(tmp_path.rglob("*.parquet"))
    assert len(parquets) == 1
    df = pl.read_parquet(parquets[0])
    assert df.height == 2
    assert "event_id" in df.columns
    assert "tenant_id" in df.columns


def test_flush_writes_parquet_for_device_event_with_struct(tmp_path: pathlib.Path):
    sink = Sink(tmp_path, batch_size=2, flush_interval_s=999)
    sink.append(device.INFO_HARDWARE_V1, _device_info_hardware_row())
    sink.append(device.INFO_HARDWARE_V1, _device_info_hardware_row())
    sink.maybe_flush()

    parquets = list(tmp_path.rglob("*.parquet"))
    assert len(parquets) == 1
    df = pl.read_parquet(parquets[0])
    assert df.height == 2
    assert "eeprom_format_version" in df.columns


def test_flush_writes_parquet_for_device_diagnostics_with_list_of_struct(tmp_path: pathlib.Path):
    sink = Sink(tmp_path, batch_size=2, flush_interval_s=999)
    sink.append(device.DIAGNOSTICS_STACK_USAGE_V1, _device_diagnostics_stack_usage_row())
    sink.append(device.DIAGNOSTICS_STACK_USAGE_V1, _device_diagnostics_stack_usage_row())
    sink.maybe_flush()

    parquets = list(tmp_path.rglob("*.parquet"))
    assert len(parquets) == 1
    df = pl.read_parquet(parquets[0])
    assert df.height == 2
    assert "threads" in df.columns


def test_partition_directory_uses_safe_name(tmp_path: pathlib.Path):
    sink = Sink(tmp_path, batch_size=2, flush_interval_s=999)
    sink.append(device.DIAGNOSTICS_STACK_USAGE_V1, _device_diagnostics_stack_usage_row())
    sink.append(device.DIAGNOSTICS_STACK_USAGE_V1, _device_diagnostics_stack_usage_row())
    sink.maybe_flush()

    dirs = [d.name for d in tmp_path.iterdir() if d.is_dir()]
    assert "device_diagnostics_stack_usage_v1" in dirs
