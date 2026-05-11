package registry

import "strings"

type Diagnostic struct {
	Location string
	Message  string
}

type Diagnostics []Diagnostic

func (d Diagnostics) HasErrors() bool {
	return len(d) > 0
}

func (d Diagnostics) Error() string {
	if len(d) == 0 {
		return ""
	}

	lines := make([]string, 0, len(d))
	for _, diagnostic := range d {
		if diagnostic.Location == "" {
			lines = append(lines, diagnostic.Message)
			continue
		}
		lines = append(lines, diagnostic.Location+": "+diagnostic.Message)
	}
	return strings.Join(lines, "\n")
}
