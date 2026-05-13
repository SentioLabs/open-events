module github.com/sentiolabs/open-events/examples/demo/services/api

go 1.24

require github.com/acme/storefront/events v0.0.0

replace github.com/acme/storefront/events => ../../../../_build/demo-proto/gen/go/com/acme/storefront/v1
