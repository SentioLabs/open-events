from consumer.attrs import ATTR_EVENT_NAME, ATTR_SCHEMA, SCHEMA_VALUE


def test_contract_attr_names():
    assert ATTR_EVENT_NAME == "event_name"
    assert ATTR_SCHEMA == "schema"


def test_contract_schema_value():
    assert SCHEMA_VALUE == "openevents:com.acme.storefront/v1"
