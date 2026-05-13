"""Canonical event-name strings.

Mirror examples/demo/services/api/eventmap/registry.go. The pb2 class
mapping lives in dispatch.py (added in T3) so this module has no
generated-code dependency.
"""

CHECKOUT_STARTED_V1 = "checkout.started@1"
CHECKOUT_COMPLETED_V1 = "checkout.completed@1"
SEARCH_PERFORMED_V1 = "search.performed@1"

ALL_EVENT_NAMES: tuple[str, ...] = (
    CHECKOUT_STARTED_V1,
    CHECKOUT_COMPLETED_V1,
    SEARCH_PERFORMED_V1,
)
