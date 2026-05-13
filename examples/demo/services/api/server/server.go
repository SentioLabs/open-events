package server

import (
	"encoding/base64"
	"net/http"

	"github.com/labstack/echo/v4"
	"google.golang.org/protobuf/proto"

	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	"github.com/sentiolabs/open-events/examples/demo/services/api/publisher"
)

// envelopeMessage is the subset of the generated proto envelope that the
// handler relies on for the response body. CheckoutStartedV1, CheckoutCompletedV1,
// and SearchPerformedV1 all satisfy this interface via their generated getters.
type envelopeMessage interface {
	proto.Message
	GetEventId() string
}

// buildFunc decodes the request body, validates it, and returns the proto
// message ready to publish. If fieldErrs is non-empty, the handler returns 400.
type buildFunc func(c echo.Context) (msg envelopeMessage, fieldErrs []eventmap.FieldError, err error)

// New wires Echo routes, healthcheck, and per-event handlers backed by pub.
func New(pub publisher.Publisher, queueURL string) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	e.POST("/v1/events/checkout-started", handle(pub, queueURL, eventmap.CheckoutStartedV1, func(c echo.Context) (envelopeMessage, []eventmap.FieldError, error) {
		var req eventmap.CheckoutStartedRequest
		if err := c.Bind(&req); err != nil {
			return nil, nil, err
		}
		if errs := req.Validate(); len(errs) > 0 {
			return nil, errs, nil
		}
		return req.ToProto(), nil, nil
	}))

	e.POST("/v1/events/checkout-completed", handle(pub, queueURL, eventmap.CheckoutCompletedV1, func(c echo.Context) (envelopeMessage, []eventmap.FieldError, error) {
		var req eventmap.CheckoutCompletedRequest
		if err := c.Bind(&req); err != nil {
			return nil, nil, err
		}
		if errs := req.Validate(); len(errs) > 0 {
			return nil, errs, nil
		}
		return req.ToProto(), nil, nil
	}))

	e.POST("/v1/events/search-performed", handle(pub, queueURL, eventmap.SearchPerformedV1, func(c echo.Context) (envelopeMessage, []eventmap.FieldError, error) {
		var req eventmap.SearchPerformedRequest
		if err := c.Bind(&req); err != nil {
			return nil, nil, err
		}
		if errs := req.Validate(); len(errs) > 0 {
			return nil, errs, nil
		}
		return req.ToProto(), nil, nil
	}))

	return e
}

func handle(pub publisher.Publisher, queueURL, eventName string, build buildFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		msg, fieldErrs, err := build(c)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
		}
		if len(fieldErrs) > 0 {
			return c.JSON(http.StatusBadRequest, map[string]any{"errors": fieldErrs})
		}
		wire, err := proto.Marshal(msg)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
		}
		body := base64.StdEncoding.EncodeToString(wire)
		attrs := map[string]string{
			publisher.AttrEventName: eventName,
			publisher.AttrSchema:    publisher.SchemaValue,
		}
		msgID, err := pub.Publish(c.Request().Context(), eventName, []byte(body), attrs)
		if err != nil {
			return c.JSON(http.StatusBadGateway, map[string]any{"error": err.Error()})
		}
		return c.JSON(http.StatusAccepted, map[string]any{
			"event_id":   msg.GetEventId(),
			"queue_url":  queueURL,
			"message_id": msgID,
		})
	}
}
