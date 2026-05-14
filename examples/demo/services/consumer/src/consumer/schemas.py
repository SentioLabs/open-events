from __future__ import annotations

import polars as pl
from com.acme.storefront.v1 import events_pb2
from google.protobuf.descriptor import FieldDescriptor

from .event_names import (
    CHECKOUT_COMPLETED_V1,
    CHECKOUT_STARTED_V1,
    SEARCH_PERFORMED_V1,
)

_SCALAR_DTYPES: dict[int, type[pl.DataType]] = {
    FieldDescriptor.TYPE_STRING: pl.Utf8,
    FieldDescriptor.TYPE_BYTES: pl.Binary,
    FieldDescriptor.TYPE_BOOL: pl.Boolean,
    FieldDescriptor.TYPE_INT32: pl.Int32,
    FieldDescriptor.TYPE_INT64: pl.Int64,
    FieldDescriptor.TYPE_UINT32: pl.UInt32,
    FieldDescriptor.TYPE_UINT64: pl.UInt64,
    FieldDescriptor.TYPE_FLOAT: pl.Float32,
    FieldDescriptor.TYPE_DOUBLE: pl.Float64,
    FieldDescriptor.TYPE_ENUM: pl.Utf8,  # MessageToDict emits enum names
}


def _field_dtype(field: FieldDescriptor) -> pl.DataType:
    if field.type == FieldDescriptor.TYPE_MESSAGE:
        if field.message_type.full_name == "google.protobuf.Timestamp":
            dtype: pl.DataType = pl.Utf8()  # MessageToDict emits RFC3339 strings
        else:
            dtype = _struct_from_descriptor(field.message_type)
    else:
        dtype_cls = _SCALAR_DTYPES.get(field.type)
        if dtype_cls is None:
            raise ValueError(f"unsupported proto type {field.type} on {field.full_name}")
        dtype = dtype_cls()
    if field.is_repeated:
        return pl.List(dtype)
    return dtype


def _struct_from_descriptor(descriptor) -> pl.Struct:
    return pl.Struct({f.name: _field_dtype(f) for f in descriptor.fields})


def _schema_for(msg_cls) -> pl.Schema:
    return pl.Schema({f.name: _field_dtype(f) for f in msg_cls.DESCRIPTOR.fields})


EVENT_SCHEMAS: dict[str, pl.Schema] = {
    CHECKOUT_STARTED_V1: _schema_for(events_pb2.CheckoutStartedV1),
    CHECKOUT_COMPLETED_V1: _schema_for(events_pb2.CheckoutCompletedV1),
    SEARCH_PERFORMED_V1: _schema_for(events_pb2.SearchPerformedV1),
}
