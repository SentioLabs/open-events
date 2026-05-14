from __future__ import annotations

import polars as pl

# ---------------------------------------------------------------------------
# Shared envelope and context columns for all user-domain events.
# ---------------------------------------------------------------------------
_ENVELOPE = {
    "event_name": pl.Utf8,
    "event_version": pl.Int32,
    "event_id": pl.Utf8,
    "event_ts": pl.Datetime("us", "UTC"),
}

_USER_CONTEXT = {
    "tenant_id": pl.Utf8,
    "user_id": pl.Utf8,
    "session_id": pl.Utf8,
    "platform": pl.Utf8,
}

# ---------------------------------------------------------------------------
# user.auth.signup@1
# ---------------------------------------------------------------------------
AUTH_SIGNUP_SCHEMA = pl.Schema({
    **_ENVELOPE,
    **_USER_CONTEXT,
    "method": pl.Utf8,
    "plan": pl.Utf8,
})

# ---------------------------------------------------------------------------
# user.auth.login@1
# ---------------------------------------------------------------------------
AUTH_LOGIN_SCHEMA = pl.Schema({
    **_ENVELOPE,
    **_USER_CONTEXT,
    "method": pl.Utf8,
    "success": pl.Boolean,
})

# ---------------------------------------------------------------------------
# user.auth.logout@1
# ---------------------------------------------------------------------------
AUTH_LOGOUT_SCHEMA = pl.Schema({
    **_ENVELOPE,
    **_USER_CONTEXT,
    "duration_seconds": pl.Int64,
})

# ---------------------------------------------------------------------------
# user.cart.checkout@1
# ---------------------------------------------------------------------------
CART_CHECKOUT_SCHEMA = pl.Schema({
    **_ENVELOPE,
    **_USER_CONTEXT,
    "cart_id": pl.Utf8,
    "item_count": pl.Int64,
    "subtotal_cents": pl.Int64,
    "currency": pl.Utf8,
})

# ---------------------------------------------------------------------------
# user.cart.purchase@1
# ---------------------------------------------------------------------------
CART_PURCHASE_SCHEMA = pl.Schema({
    **_ENVELOPE,
    **_USER_CONTEXT,
    "cart_id": pl.Utf8,
    "order_id": pl.Utf8,
    "total_cents": pl.Int64,
    "payment_method": pl.Utf8,
    "coupon_code": pl.Utf8,
})

# ---------------------------------------------------------------------------
# user.cart.item_added@1
# ---------------------------------------------------------------------------
CART_ITEM_ADDED_SCHEMA = pl.Schema({
    **_ENVELOPE,
    **_USER_CONTEXT,
    "cart_id": pl.Utf8,
    "sku": pl.Utf8,
    "quantity": pl.Int64,
})
