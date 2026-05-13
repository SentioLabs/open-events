from consumer.event_names import (
    ALL_EVENT_NAMES,
    CHECKOUT_COMPLETED_V1,
    CHECKOUT_STARTED_V1,
    SEARCH_PERFORMED_V1,
)


def test_contract_event_names():
    assert CHECKOUT_STARTED_V1 == "checkout.started@1"
    assert CHECKOUT_COMPLETED_V1 == "checkout.completed@1"
    assert SEARCH_PERFORMED_V1 == "search.performed@1"


def test_all_event_names_order():
    assert ALL_EVENT_NAMES == (
        "checkout.started@1",
        "checkout.completed@1",
        "search.performed@1",
    )
