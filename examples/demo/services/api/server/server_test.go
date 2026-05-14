package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	devicepb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/device"
	userpb "github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/pb/user"
	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/user"
	"github.com/sentiolabs/open-events/examples/demo/services/api/publisher"
)

const testQueueURL = "http://localhost:4566/000000000000/test-queue"

func newTestServer(t *testing.T) (*publisher.FakePublisher, http.Handler) {
	t.Helper()
	pub := &publisher.FakePublisher{}
	e := New(pub, testQueueURL)
	return pub, e
}

func TestHealthz(t *testing.T) {
	_, h := newTestServer(t)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status: got %d want 200", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != "ok" {
		t.Errorf("body: got %q want %q", got, "ok")
	}
}

// --- User routes ---

func TestPostUserAuthSignup_Publishes(t *testing.T) {
	pub, h := newTestServer(t)

	payload := map[string]any{
		"context": map[string]any{
			"tenant_id": "tenant-1",
			"user_id":   "user-1",
			"platform":  "web",
		},
		"method": "email",
		"plan":   "pro",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/events/user/auth/signup", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status: got %d want 202; body=%s", rec.Code, rec.Body.String())
	}
	if len(pub.Calls) != 1 {
		t.Fatalf("Calls: got %d want 1", len(pub.Calls))
	}
	call := pub.Calls[0]
	if call.Attrs[publisher.AttrEventName] != user.AuthSignupV1 {
		t.Errorf("AttrEventName: got %q want %q", call.Attrs[publisher.AttrEventName], user.AuthSignupV1)
	}

	wire, err := base64.StdEncoding.DecodeString(call.Body)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	var msg userpb.UserAuthSignupV1
	if err := proto.Unmarshal(wire, &msg); err != nil {
		t.Fatalf("proto.Unmarshal: %v", err)
	}
	if msg.GetEventName() != user.AuthSignupV1 {
		t.Errorf("EventName: %q", msg.GetEventName())
	}
	if _, err := uuid.Parse(msg.GetEventId()); err != nil {
		t.Errorf("EventId %q is not a valid uuid: %v", msg.GetEventId(), err)
	}
	if msg.GetContext().GetTenantId() != "tenant-1" {
		t.Errorf("Context.TenantId: %q", msg.GetContext().GetTenantId())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["queue_url"] != testQueueURL {
		t.Errorf("queue_url: %v", resp["queue_url"])
	}
}

func TestPostUserAuthSignup_RejectsMissingTenantID(t *testing.T) {
	_, h := newTestServer(t)
	body := `{"context":{"platform":"web"},"method":"email"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/events/user/auth/signup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d want 400; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "context.tenant_id") {
		t.Errorf("expected context.tenant_id in error body; got %s", rec.Body.String())
	}
}

func TestPostUserAuthSignup_RejectsBadMethod(t *testing.T) {
	_, h := newTestServer(t)
	body := `{"context":{"tenant_id":"t","platform":"web"},"method":"fax"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/events/user/auth/signup", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d want 400; body=%s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "method") {
		t.Errorf("expected method in error body; got %s", rec.Body.String())
	}
}

func TestPostUserCartCheckout_Publishes(t *testing.T) {
	pub, h := newTestServer(t)

	payload := map[string]any{
		"context": map[string]any{
			"tenant_id": "tenant-1",
			"platform":  "ios",
		},
		"cart_id":        "cart-1",
		"item_count":     3,
		"subtotal_cents": 9999,
		"currency":       "USD",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/events/user/cart/checkout", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status: got %d want 202; body=%s", rec.Code, rec.Body.String())
	}
	if len(pub.Calls) != 1 {
		t.Fatalf("Calls: got %d want 1", len(pub.Calls))
	}
	if pub.Calls[0].Attrs[publisher.AttrEventName] != user.CartCheckoutV1 {
		t.Errorf("AttrEventName: %q", pub.Calls[0].Attrs[publisher.AttrEventName])
	}

	wire, err := base64.StdEncoding.DecodeString(pub.Calls[0].Body)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	var msg userpb.UserCartCheckoutV1
	if err := proto.Unmarshal(wire, &msg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if msg.GetProperties().GetCartId() != "cart-1" {
		t.Errorf("CartId: %q", msg.GetProperties().GetCartId())
	}
	if msg.GetProperties().GetItemCount() != 3 {
		t.Errorf("ItemCount: %d", msg.GetProperties().GetItemCount())
	}
}

func TestPostUserCartCheckout_RejectsMissingCartID(t *testing.T) {
	_, h := newTestServer(t)
	body := `{"context":{"tenant_id":"t","platform":"web"},"item_count":1,"subtotal_cents":1,"currency":"USD"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/events/user/cart/checkout", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "cart_id") {
		t.Errorf("expected cart_id in error body; got %s", rec.Body.String())
	}
}

func TestPostUserCartPurchase_Publishes(t *testing.T) {
	pub, h := newTestServer(t)
	body := `{"context":{"tenant_id":"t","platform":"ios"},"cart_id":"c","order_id":"o","total_cents":500,"payment_method":"card"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/events/user/cart/purchase", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status: got %d want 202; body=%s", rec.Code, rec.Body.String())
	}
	if pub.Calls[0].Attrs[publisher.AttrEventName] != user.CartPurchaseV1 {
		t.Errorf("AttrEventName: %q", pub.Calls[0].Attrs[publisher.AttrEventName])
	}

	wire, err := base64.StdEncoding.DecodeString(pub.Calls[0].Body)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	var msg userpb.UserCartPurchaseV1
	if err := proto.Unmarshal(wire, &msg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if msg.GetProperties().GetOrderId() != "o" {
		t.Errorf("OrderId: %q", msg.GetProperties().GetOrderId())
	}
}

func TestPostUserCartPurchase_RejectsBadPaymentMethod(t *testing.T) {
	_, h := newTestServer(t)
	body := `{"context":{"tenant_id":"t","platform":"web"},"cart_id":"c","order_id":"o","total_cents":1,"payment_method":"bitcoin"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/events/user/cart/purchase", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "payment_method") {
		t.Errorf("expected payment_method in error body; got %s", rec.Body.String())
	}
}

// --- Device routes ---

func TestPostDeviceInfoHardware_Publishes(t *testing.T) {
	pub, h := newTestServer(t)

	payload := map[string]any{
		"context": map[string]any{
			"tenant_id": "tenant-1",
			"device_id": "device-1",
		},
		"unique_id":               "abc123",
		"manufacturing_timestamp": "2024-01-01T00:00:00Z",
		"eeprom_format_version":   map[string]any{"major": 1, "minor": 0},
		"module_pcb_version":      map[string]any{"major": 2, "minor": 1},
		"sensor_type":             "co",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/events/device/info/hardware", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status: got %d want 202; body=%s", rec.Code, rec.Body.String())
	}

	wire, err := base64.StdEncoding.DecodeString(pub.Calls[0].Body)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	var msg devicepb.DeviceInfoHardwareV1
	if err := proto.Unmarshal(wire, &msg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if msg.GetProperties().GetUniqueId() != "abc123" {
		t.Errorf("UniqueId: %q", msg.GetProperties().GetUniqueId())
	}
	if msg.GetProperties().GetSensorType() != devicepb.DeviceInfoHardwareV1Properties_SENSOR_TYPE_CO {
		t.Errorf("SensorType: %v", msg.GetProperties().GetSensorType())
	}
}

func TestPostDeviceInfoHardware_RejectsBadSensorType(t *testing.T) {
	_, h := newTestServer(t)
	body := `{"context":{"tenant_id":"t","device_id":"d"},"unique_id":"u","eeprom_format_version":{"major":1,"minor":0},"module_pcb_version":{"major":1,"minor":0},"sensor_type":"invalid"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/events/device/info/hardware", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d want 400", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "sensor_type") {
		t.Errorf("expected sensor_type in error body; got %s", rec.Body.String())
	}
}

func TestPostDeviceIncidentDrop_Publishes(t *testing.T) {
	pub, h := newTestServer(t)
	body := `{"context":{"tenant_id":"t","device_id":"d"},"peak_acceleration_g":3.5,"axis":"z","duration_ms":120}`
	req := httptest.NewRequest(http.MethodPost, "/v1/events/device/incident/drop", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status: got %d want 202; body=%s", rec.Code, rec.Body.String())
	}

	wire, err := base64.StdEncoding.DecodeString(pub.Calls[0].Body)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	var msg devicepb.DeviceIncidentDropV1
	if err := proto.Unmarshal(wire, &msg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if msg.GetProperties().GetAxis() != devicepb.DeviceIncidentDropV1Properties_AXIS_Z {
		t.Errorf("Axis: %v", msg.GetProperties().GetAxis())
	}
}

func TestPostDeviceDiagnosticsStackUsage_Publishes(t *testing.T) {
	pub, h := newTestServer(t)
	payload := map[string]any{
		"context": map[string]any{
			"tenant_id": "tenant-1",
			"device_id": "device-1",
		},
		"thread_count":          2,
		"highest_usage_percent": 75,
		"highest_usage_thread":  "main",
		"threads": []map[string]any{
			{
				"name":             "main",
				"stack_size_bytes": 4096,
				"stack_used_bytes": 3072,
				"usage_percent":    75,
				"priority":         5,
				"state":            "running",
			},
		},
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/v1/events/device/diagnostics/stack_usage", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status: got %d want 202; body=%s", rec.Code, rec.Body.String())
	}

	wire, err := base64.StdEncoding.DecodeString(pub.Calls[0].Body)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	var msg devicepb.DeviceDiagnosticsStackUsageV1
	if err := proto.Unmarshal(wire, &msg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if msg.GetProperties().GetHighestUsageThread() != "main" {
		t.Errorf("HighestUsageThread: %q", msg.GetProperties().GetHighestUsageThread())
	}
	if len(msg.GetProperties().GetThreads()) != 1 {
		t.Errorf("Threads count: %d", len(msg.GetProperties().GetThreads()))
	}
}

// --- Route count assertion ---

func TestServer_Registers12Routes(t *testing.T) {
	// Count POST routes registered by checking each expected path returns not-404.
	_, h := newTestServer(t)

	paths := []string{
		"/v1/events/user/auth/signup",
		"/v1/events/user/auth/login",
		"/v1/events/user/auth/logout",
		"/v1/events/user/cart/checkout",
		"/v1/events/user/cart/purchase",
		"/v1/events/user/cart/item_added",
		"/v1/events/device/info/hardware",
		"/v1/events/device/info/software",
		"/v1/events/device/info/calibration",
		"/v1/events/device/incident/temperature",
		"/v1/events/device/incident/drop",
		"/v1/events/device/diagnostics/stack_usage",
	}

	for _, path := range paths {
		// POST with empty body — we expect 400 (validation error) NOT 404 (route not found).
		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)

		if rec.Code == http.StatusNotFound {
			t.Errorf("route not registered: POST %s returned 404", path)
		}
	}
}
