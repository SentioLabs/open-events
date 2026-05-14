package eventmap_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/device"
	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/user"
)

// --- BindBuild ---

type fakeRequest struct {
	Name string `json:"name"`
	fail bool
}

func (r fakeRequest) Validate() []eventmap.FieldError {
	if r.Name == "" {
		return []eventmap.FieldError{{Field: "name", Message: "required"}}
	}
	return nil
}

func TestBindBuild_ReturnsFieldErrors_WhenValidationFails(t *testing.T) {
	e := echo.New()
	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	_, errs, err := eventmap.BindBuild[fakeRequest](c, func(r fakeRequest) eventmap.EnvelopeMessage {
		called = true
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("toProto should not be called when validation fails")
	}
	if len(errs) == 0 {
		t.Error("expected validation errors, got none")
	}
	if errs[0].Field != "name" {
		t.Errorf("expected field=name, got %q", errs[0].Field)
	}
}

func TestBindBuild_ReturnsBindError_WhenBodyMalformed(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`not json`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	_, _, err := eventmap.BindBuild[fakeRequest](c, func(r fakeRequest) eventmap.EnvelopeMessage {
		return nil
	})
	if err == nil {
		t.Error("expected bind error for malformed JSON, got nil")
	}
}

// --- user.Routes ---

func TestUserRoutes_Returns6Routes(t *testing.T) {
	routes := user.Routes()
	if len(routes) != 6 {
		t.Fatalf("expected 6 user routes, got %d", len(routes))
	}
}

func TestUserRoutes_AllPathsMatchConvention(t *testing.T) {
	for _, r := range user.Routes() {
		if !strings.HasPrefix(r.Path, "/v1/events/user/") {
			t.Errorf("route path %q does not start with /v1/events/user/", r.Path)
		}
		if strings.Contains(r.Path, "-") {
			t.Errorf("route path %q uses kebab-case; must use slash-separated segments", r.Path)
		}
	}
}

func TestUserRoutes_ContainsExpectedPaths(t *testing.T) {
	routes := user.Routes()
	paths := make(map[string]bool, len(routes))
	for _, r := range routes {
		paths[r.Path] = true
	}

	want := []string{
		"/v1/events/user/auth/signup",
		"/v1/events/user/auth/login",
		"/v1/events/user/auth/logout",
		"/v1/events/user/cart/checkout",
		"/v1/events/user/cart/purchase",
		"/v1/events/user/cart/item_added",
	}
	for _, p := range want {
		if !paths[p] {
			t.Errorf("missing expected route: %s", p)
		}
	}
}

func TestUserRoutes_AllRoutesHaveEventName(t *testing.T) {
	for _, r := range user.Routes() {
		if r.EventName == "" {
			t.Errorf("route %s has empty EventName", r.Path)
		}
	}
}

func TestUserRoutes_AllRoutesHaveBuildFunc(t *testing.T) {
	for _, r := range user.Routes() {
		if r.Build == nil {
			t.Errorf("route %s has nil Build func", r.Path)
		}
	}
}

// --- device.Routes ---

func TestDeviceRoutes_Returns6Routes(t *testing.T) {
	routes := device.Routes()
	if len(routes) != 6 {
		t.Fatalf("expected 6 device routes, got %d", len(routes))
	}
}

func TestDeviceRoutes_AllPathsMatchConvention(t *testing.T) {
	for _, r := range device.Routes() {
		if !strings.HasPrefix(r.Path, "/v1/events/device/") {
			t.Errorf("route path %q does not start with /v1/events/device/", r.Path)
		}
		if strings.Contains(r.Path, "-") {
			t.Errorf("route path %q uses kebab-case; must use slash-separated segments", r.Path)
		}
	}
}

func TestDeviceRoutes_ContainsExpectedPaths(t *testing.T) {
	routes := device.Routes()
	paths := make(map[string]bool, len(routes))
	for _, r := range routes {
		paths[r.Path] = true
	}

	want := []string{
		"/v1/events/device/info/hardware",
		"/v1/events/device/info/software",
		"/v1/events/device/info/calibration",
		"/v1/events/device/incident/temperature",
		"/v1/events/device/incident/drop",
		"/v1/events/device/diagnostics/stack_usage",
	}
	for _, p := range want {
		if !paths[p] {
			t.Errorf("missing expected route: %s", p)
		}
	}
}

func TestDeviceRoutes_AllRoutesHaveEventName(t *testing.T) {
	for _, r := range device.Routes() {
		if r.EventName == "" {
			t.Errorf("route %s has empty EventName", r.Path)
		}
	}
}

func TestDeviceRoutes_AllRoutesHaveBuildFunc(t *testing.T) {
	for _, r := range device.Routes() {
		if r.Build == nil {
			t.Errorf("route %s has nil Build func", r.Path)
		}
	}
}
