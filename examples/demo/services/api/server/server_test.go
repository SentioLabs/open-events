package server

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	eventspb "github.com/acme/storefront/events"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
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

func TestPostCheckoutStarted_Publishes(t *testing.T) {
	pub, h := newTestServer(t)

	payload := map[string]any{
		"context": map[string]any{
			"tenant_id":  "tenant-1",
			"user_id":    "user-1",
			"session_id": "session-1",
			"platform":   "WEB",
		},
		"cart_id":        "cart-1",
		"item_count":     3,
		"subtotal_cents": 9999,
		"currency":       "USD",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/v1/events/checkout-started", strings.NewReader(string(body)))
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
	if call.Attrs[publisher.AttrEventName] != eventmap.CheckoutStartedV1 {
		t.Errorf("AttrEventName: got %q want %q", call.Attrs[publisher.AttrEventName], eventmap.CheckoutStartedV1)
	}
	if call.Attrs[publisher.AttrSchema] != publisher.SchemaValue {
		t.Errorf("AttrSchema: got %q want %q", call.Attrs[publisher.AttrSchema], publisher.SchemaValue)
	}

	// Body is base64-encoded protobuf; decode and assert fields round-trip.
	wire, err := base64.StdEncoding.DecodeString(call.Body)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	var msg eventspb.CheckoutStartedV1
	if err := proto.Unmarshal(wire, &msg); err != nil {
		t.Fatalf("proto.Unmarshal: %v", err)
	}
	if msg.GetProperties().GetCartId() != "cart-1" {
		t.Errorf("CartId: %q", msg.GetProperties().GetCartId())
	}
	if msg.GetProperties().GetItemCount() != 3 {
		t.Errorf("ItemCount: %d", msg.GetProperties().GetItemCount())
	}
	if msg.GetContext().GetTenantId() != "tenant-1" {
		t.Errorf("Context.TenantId: %q", msg.GetContext().GetTenantId())
	}

	// Response body has event_id (valid UUID), queue_url, message_id.
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	eventID, _ := resp["event_id"].(string)
	if _, err := uuid.Parse(eventID); err != nil {
		t.Errorf("event_id %q is not a valid uuid: %v", eventID, err)
	}
	if eventID != msg.GetEventId() {
		t.Errorf("response event_id %q != proto EventId %q", eventID, msg.GetEventId())
	}
	if resp["queue_url"] != testQueueURL {
		t.Errorf("queue_url: %v", resp["queue_url"])
	}
	if resp["message_id"] != "fake-msg-id" {
		t.Errorf("message_id: %v", resp["message_id"])
	}
}

func TestPostCheckoutStarted_RejectsMissingTenantID(t *testing.T) {
	pub, h := newTestServer(t)

	body := `{"context":{"platform":"WEB"},"cart_id":"c","item_count":1,"subtotal_cents":1,"currency":"USD"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/events/checkout-started", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d want 400; body=%s", rec.Code, rec.Body.String())
	}
	if len(pub.Calls) != 0 {
		t.Errorf("publisher should not be called on validation error, got %d calls", len(pub.Calls))
	}
	if !strings.Contains(rec.Body.String(), "context.tenant_id") {
		t.Errorf("expected context.tenant_id in error body; got %s", rec.Body.String())
	}
}

func TestPostCheckoutCompleted_Publishes(t *testing.T) {
	pub, h := newTestServer(t)
	body := `{"context":{"tenant_id":"t","platform":"IOS"},"cart_id":"c","order_id":"o","total_cents":500,"payment_method":"CARD"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/events/checkout-completed", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status: got %d want 202; body=%s", rec.Code, rec.Body.String())
	}
	if len(pub.Calls) != 1 {
		t.Fatalf("Calls: got %d want 1", len(pub.Calls))
	}
	if pub.Calls[0].Attrs[publisher.AttrEventName] != eventmap.CheckoutCompletedV1 {
		t.Errorf("AttrEventName: %q", pub.Calls[0].Attrs[publisher.AttrEventName])
	}

	wire, err := base64.StdEncoding.DecodeString(pub.Calls[0].Body)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	var msg eventspb.CheckoutCompletedV1
	if err := proto.Unmarshal(wire, &msg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if msg.GetProperties().GetOrderId() != "o" {
		t.Errorf("OrderId: %q", msg.GetProperties().GetOrderId())
	}
}

func TestPostCheckoutCompleted_RejectsBadPaymentMethod(t *testing.T) {
	pub, h := newTestServer(t)
	body := `{"context":{"tenant_id":"t","platform":"WEB"},"cart_id":"c","order_id":"o","total_cents":1,"payment_method":"BITCOIN"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/events/checkout-completed", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d want 400", rec.Code)
	}
	if len(pub.Calls) != 0 {
		t.Errorf("publisher should not be called on validation error")
	}
	if !strings.Contains(rec.Body.String(), "payment_method") {
		t.Errorf("expected payment_method in error body; got %s", rec.Body.String())
	}
}

func TestPostSearchPerformed_Publishes(t *testing.T) {
	pub, h := newTestServer(t)
	body := `{"context":{"tenant_id":"t","platform":"ANDROID"},"query":"shoes","result_count":5,"filters":["brand:acme"]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/events/search-performed", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("status: got %d want 202; body=%s", rec.Code, rec.Body.String())
	}
	if pub.Calls[0].Attrs[publisher.AttrEventName] != eventmap.SearchPerformedV1 {
		t.Errorf("AttrEventName: %q", pub.Calls[0].Attrs[publisher.AttrEventName])
	}

	wire, err := base64.StdEncoding.DecodeString(pub.Calls[0].Body)
	if err != nil {
		t.Fatalf("base64 decode: %v", err)
	}
	var msg eventspb.SearchPerformedV1
	if err := proto.Unmarshal(wire, &msg); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if msg.GetProperties().GetQuery() != "shoes" {
		t.Errorf("Query: %q", msg.GetProperties().GetQuery())
	}
	if got := msg.GetProperties().GetFilters(); len(got) != 1 || got[0] != "brand:acme" {
		t.Errorf("Filters: %+v", got)
	}
}

func TestPostSearchPerformed_RejectsMissingQuery(t *testing.T) {
	pub, h := newTestServer(t)
	body := `{"context":{"tenant_id":"t","platform":"WEB"},"result_count":0}`
	req := httptest.NewRequest(http.MethodPost, "/v1/events/search-performed", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status: got %d want 400", rec.Code)
	}
	if len(pub.Calls) != 0 {
		t.Errorf("publisher should not be called on validation error")
	}
	if !strings.Contains(rec.Body.String(), "query") {
		t.Errorf("expected query in error body; got %s", rec.Body.String())
	}
}
