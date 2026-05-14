package schemair

import (
	"sort"

	"github.com/sentiolabs/open-events/internal/registry"
)

// DomainBundle groups a Domain with all events that belong to it.
type DomainBundle struct {
	Domain registry.Domain
	Events []registry.Event
}

// GroupByDomain partitions the events in reg by their Domain field and returns
// one DomainBundle per domain, ordered alphabetically by domain name.
func GroupByDomain(reg registry.Registry) []DomainBundle {
	names := make([]string, 0, len(reg.Domains))
	for name := range reg.Domains {
		names = append(names, name)
	}
	sort.Strings(names)
	bundles := make([]DomainBundle, 0, len(names))
	for _, name := range names {
		d := reg.Domains[name]
		events := make([]registry.Event, 0)
		for _, e := range reg.Events {
			if e.Domain == name {
				events = append(events, e)
			}
		}
		bundles = append(bundles, DomainBundle{Domain: d, Events: events})
	}
	return bundles
}
