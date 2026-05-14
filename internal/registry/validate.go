package registry

import (
	"fmt"
	"go/token"
	"regexp"
	"sort"
	"strings"
)

var (
	snakeCasePattern     = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	goPackagePattern     = regexp.MustCompile(`^[a-z0-9]+([._/-][a-z0-9]+)*$`)
	pythonPackagePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$`)
)

// pythonKeywords lists Python 3.11+ reserved words that would collide with a
// domain name used as a module identifier (e.g. `from acme.events.class import ...`
// is a syntax error). The list mirrors Python's `keyword.kwlist`.
var pythonKeywords = map[string]struct{}{
	"False": {}, "None": {}, "True": {}, "and": {}, "as": {}, "assert": {},
	"async": {}, "await": {}, "break": {}, "class": {}, "continue": {},
	"def": {}, "del": {}, "elif": {}, "else": {}, "except": {}, "finally": {},
	"for": {}, "from": {}, "global": {}, "if": {}, "import": {}, "in": {},
	"is": {}, "lambda": {}, "nonlocal": {}, "not": {}, "or": {}, "pass": {},
	"raise": {}, "return": {}, "try": {}, "while": {}, "with": {}, "yield": {},
	"match": {}, "case": {}, // soft keywords as of 3.10+
}

// Validate checks the registry for structural, referential, uniqueness, and
// field-level errors. It expects reg to have been produced by Load.
func Validate(reg Registry) Diagnostics {
	var diags Diagnostics

	// Legacy top-level fields (still populated from openevents.yaml via Load).
	if strings.TrimSpace(reg.Version) != "" || strings.TrimSpace(reg.Namespace) != "" ||
		reg.Package.Go != "" || reg.Package.Python != "" {
		validateTopLevel(reg, &diags)
	}

	// Structural: domains must be non-empty.
	if len(reg.Domains) == 0 {
		diags = append(diags, Diagnostic{Location: "registry", Message: "registry must define at least one domain"})
	}

	// Build set of known owner slugs for referential checks.
	ownerSlugs := make(map[string]struct{}, len(reg.Owners))
	for _, o := range reg.Owners {
		ownerSlugs[o.Team] = struct{}{}
	}

	// Validate domains.
	for _, domainName := range sortedStringKeys(reg.Domains) {
		domain := reg.Domains[domainName]
		validateDomain(domainName, domain, ownerSlugs, &diags)
	}

	// Validate events: structural depth + snake_case, referential owner, uniqueness, field-level.
	seen := make(map[string]struct{}, len(reg.Events))
	for _, event := range reg.Events {
		validateEvent(event, ownerSlugs, seen, &diags)
	}

	return diags
}

// validateTopLevel checks Version, Namespace, and Package when those fields are set
// (legacy flat-file form or loaded from openevents.yaml).
func validateTopLevel(reg Registry, diags *Diagnostics) {
	if strings.TrimSpace(reg.Version) == "" {
		*diags = append(*diags, Diagnostic{Location: "openevents", Message: "openevents is required"})
	} else if reg.Version != SupportedVersion {
		*diags = append(*diags, Diagnostic{Location: "openevents", Message: fmt.Sprintf("unsupported openevents version %q", reg.Version)})
	}
	if strings.TrimSpace(reg.Namespace) == "" {
		*diags = append(*diags, Diagnostic{Location: "namespace", Message: "namespace is required"})
	}
	validatePackages(reg.Package, diags)
}

func validatePackages(pkg PackageConfig, diags *Diagnostics) {
	if pkg.Go != "" {
		if !goPackagePattern.MatchString(pkg.Go) {
			*diags = append(*diags, Diagnostic{Location: "package.go", Message: "package.go must be a valid Go import path"})
		} else if !strings.Contains(pkg.Go, ".") && !strings.Contains(pkg.Go, "/") {
			*diags = append(*diags, Diagnostic{Location: "package.go", Message: "package.go must include at least one '.' or '/' in the import path"})
		} else {
			parts := strings.Split(pkg.Go, "/")
			base := parts[len(parts)-1]
			if token.Lookup(base).IsKeyword() {
				*diags = append(*diags, Diagnostic{Location: "package.go", Message: "package.go basename must not be a Go keyword"})
			}
		}
	}
	if pkg.Python != "" && !pythonPackagePattern.MatchString(pkg.Python) {
		*diags = append(*diags, Diagnostic{Location: "package.python", Message: "package.python must be a valid Python package name"})
	}
}

// validateDomain checks referential owner and domain-level context fields.
// The domain name itself is also checked against Go and Python keyword lists
// because the per-domain codegen uses it directly as a Go package and Python
// module identifier.
func validateDomain(domainName string, domain Domain, ownerSlugs map[string]struct{}, diags *Diagnostics) {
	domainFile := domainName + "/domain.yml"
	if token.Lookup(domainName).IsKeyword() {
		*diags = append(*diags, Diagnostic{
			Location: domainFile,
			Message:  fmt.Sprintf("domain name %q is a Go keyword and would produce uncompilable Go bindings", domainName),
		})
	}
	if _, ok := pythonKeywords[domainName]; ok {
		*diags = append(*diags, Diagnostic{
			Location: domainFile,
			Message:  fmt.Sprintf("domain name %q is a Python keyword and would produce unimportable Python modules", domainName),
		})
	}
	if domain.Owner != "" {
		if _, ok := ownerSlugs[domain.Owner]; !ok {
			*diags = append(*diags, Diagnostic{
				Location: domainFile + ":owner",
				Message:  fmt.Sprintf("owner %q is not declared in the registry owners list", domain.Owner),
			})
		}
	}
	for _, fieldName := range sortedFieldKeys(domain.Context) {
		validateField(domainFile+":context."+fieldName, domain.Context[fieldName], diags)
	}
}

// validateEvent checks structural depth, snake_case path segments, referential owner,
// uniqueness, and field-level properties.
func validateEvent(event Event, ownerSlugs map[string]struct{}, seen map[string]struct{}, diags *Diagnostics) {
	// Derive the file path from Path segments + event action (last segment of Name).
	filePath := eventFilePath(event)

	// Structural: snake_case path segments.
	for _, seg := range event.Path {
		if !snakeCasePattern.MatchString(seg) {
			*diags = append(*diags, Diagnostic{
				Location: filePath,
				Message:  fmt.Sprintf("path segment %q must be snake_case (^[a-z][a-z0-9_]*$)", seg),
			})
		}
	}

	// Structural: depth 2–4 (Path length must be 2–4).
	// Path = [domain, ...categories], so len(Path) represents the directory depth.
	// With the action name, the composed name has len(Path)+1 segments.
	// Allowed: Path len 2–4 means 3–5 name segments, but per spec: depth 2–4 means Path length 2–4.
	if len(event.Path) < 2 {
		*diags = append(*diags, Diagnostic{
			Location: filePath,
			Message:  fmt.Sprintf("event path depth %d is below minimum of 2 (need at least domain/category/action)", len(event.Path)),
		})
	} else if len(event.Path) > 4 {
		*diags = append(*diags, Diagnostic{
			Location: filePath,
			Message:  fmt.Sprintf("event path depth %d exceeds maximum of 4", len(event.Path)),
		})
	}

	// Structural: version must be positive.
	if event.Version <= 0 {
		*diags = append(*diags, Diagnostic{
			Location: filePath + ":version",
			Message:  "event version must be positive",
		})
	}

	// Structural: status must be a supported value.
	if !isSupportedStatus(event.Status) {
		*diags = append(*diags, Diagnostic{
			Location: filePath + ":status",
			Message:  fmt.Sprintf("unsupported event status %q (must be active, deprecated, or experimental)", event.Status),
		})
	}

	// Structural: action segment (last name part derived from filename) must be snake_case.
	nameParts := strings.Split(event.Name, ".")
	action := nameParts[len(nameParts)-1]
	if !snakeCasePattern.MatchString(action) {
		*diags = append(*diags, Diagnostic{
			Location: filePath + ":name",
			Message:  fmt.Sprintf("action segment %q must be snake_case (^[a-z][a-z0-9_]*$)", action),
		})
	}

	// Referential: event-level owner (optional).
	if event.Owner != "" {
		if _, ok := ownerSlugs[event.Owner]; !ok {
			*diags = append(*diags, Diagnostic{
				Location: filePath + ":owner",
				Message:  fmt.Sprintf("owner %q is not declared in the registry owners list", event.Owner),
			})
		}
	}

	// Uniqueness: composed name + version.
	key := fmt.Sprintf("%s@%d", event.Name, event.Version)
	if _, exists := seen[key]; exists {
		*diags = append(*diags, Diagnostic{
			Location: key,
			Message:  fmt.Sprintf("duplicate event name/version %q", key),
		})
	} else {
		seen[key] = struct{}{}
	}

	// Field-level validation for event properties.
	for _, name := range sortedFieldKeys(event.Properties) {
		validateField(filePath+":properties."+name, event.Properties[name], diags)
	}
}

// eventFilePath derives the relative file path for an event from its Path and Name.
// e.g. Path=["user","auth"], Name="user.auth.signup" → "user/auth/signup.yml"
func eventFilePath(event Event) string {
	if len(event.Path) == 0 {
		return event.Name + ".yml"
	}
	// Action name is the last segment of event.Name.
	nameParts := strings.Split(event.Name, ".")
	action := nameParts[len(nameParts)-1]
	dirPath := strings.Join(event.Path, "/")
	return dirPath + "/" + action + ".yml"
}

func validateField(location string, field Field, diags *Diagnostics) {
	if !snakeCasePattern.MatchString(field.Name) && field.Name != "items" {
		*diags = append(*diags, Diagnostic{Location: location, Message: "field name must be snake_case"})
	}
	if !isSupportedFieldType(field.Type) {
		*diags = append(*diags, Diagnostic{Location: location + ".type", Message: fmt.Sprintf("unsupported field type %q", field.Type)})
	}
	if !isSupportedPII(field.PII) {
		*diags = append(*diags, Diagnostic{Location: location + ".pii", Message: fmt.Sprintf("unsupported pii classification %q", field.PII)})
	}

	switch field.Type {
	case FieldTypeEnum:
		validateEnum(location, field.Values, diags)
	case FieldTypeArray:
		if field.Items == nil {
			*diags = append(*diags, Diagnostic{Location: location + ".items", Message: "array fields must define items"})
		}
	case FieldTypeObject:
		if len(field.Properties) == 0 {
			*diags = append(*diags, Diagnostic{Location: location + ".properties", Message: "object fields must define properties"})
		}
	}

	if field.Items != nil {
		validateField(location+".items", *field.Items, diags)
	}
	for _, name := range sortedFieldKeys(field.Properties) {
		validateField(location+".properties."+name, field.Properties[name], diags)
	}
}

func validateEnum(location string, values []string, diags *Diagnostics) {
	if len(values) == 0 {
		*diags = append(*diags, Diagnostic{Location: location + ".values", Message: "enum fields must define at least one value"})
		return
	}

	seen := make(map[string]struct{}, len(values))
	for index, value := range values {
		valueLocation := fmt.Sprintf("%s.values[%d]", location, index)
		if strings.TrimSpace(value) == "" {
			*diags = append(*diags, Diagnostic{Location: valueLocation, Message: "enum values must not be empty"})
			continue
		}
		if _, exists := seen[value]; exists {
			*diags = append(*diags, Diagnostic{Location: valueLocation, Message: fmt.Sprintf("duplicate enum value %q", value)})
			continue
		}
		seen[value] = struct{}{}
	}
}

func isSupportedStatus(status string) bool {
	switch status {
	case "active", "deprecated", "experimental":
		return true
	default:
		return false
	}
}

func isSupportedFieldType(fieldType FieldType) bool {
	switch fieldType {
	case FieldTypeString,
		FieldTypeInteger,
		FieldTypeNumber,
		FieldTypeBoolean,
		FieldTypeTimestamp,
		FieldTypeDate,
		FieldTypeUUID,
		FieldTypeEnum,
		FieldTypeObject,
		FieldTypeArray:
		return true
	default:
		return false
	}
}

func isSupportedPII(pii PIIClassification) bool {
	switch pii {
	case PIINone, PIIPseudonymous, PIIPersonal, PIISensitive:
		return true
	default:
		return false
	}
}

func sortedFieldKeys(fields map[string]Field) []string {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedStringKeys(m map[string]Domain) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
