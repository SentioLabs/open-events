package schemair

import (
	"testing"

	"github.com/sentiolabs/open-events/internal/registry"
)

var _ func(registry.Registry) []DomainBundle = GroupByDomain
var _ registry.Domain = DomainBundle{}.Domain
var _ []registry.Event = DomainBundle{}.Events

func TestGroupByDomain_PartitionsAndOrdersAlphabetically(t *testing.T) {
	reg := registry.Registry{
		Domains: map[string]registry.Domain{
			"user":   {Name: "user"},
			"device": {Name: "device"},
		},
		Events: []registry.Event{
			{Name: "user.auth.signup", Domain: "user"},
			{Name: "device.info.hardware", Domain: "device"},
			{Name: "user.cart.checkout", Domain: "user"},
		},
	}
	bundles := GroupByDomain(reg)
	if len(bundles) != 2 || bundles[0].Domain.Name != "device" || bundles[1].Domain.Name != "user" {
		t.Fatalf("unexpected ordering/count: %+v", bundles)
	}
	if len(bundles[0].Events) != 1 || len(bundles[1].Events) != 2 {
		t.Fatalf("unexpected event counts: %+v", bundles)
	}
}
