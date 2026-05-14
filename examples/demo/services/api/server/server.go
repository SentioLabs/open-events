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

type route struct {
	path      string
	eventName string
	build     buildFunc
}

type errorResponse struct {
	Error string `json:"error"`
}

type fieldErrorsResponse struct {
	Errors []eventmap.FieldError `json:"errors"`
}

type acceptedResponse struct {
	EventID   string `json:"event_id"`
	QueueURL  string `json:"queue_url"`
	MessageID string `json:"message_id"`
}

// New wires Echo routes, healthcheck, and per-event handlers backed by pub.
func New(pub publisher.Publisher, queueURL string) *echo.Echo {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.GET("/healthz", func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	for _, r := range routes() {
		e.POST(r.path, handle(pub, queueURL, r.eventName, r.build))
	}

	return e
}

func routes() []route {
	return []route{
		{
			path:      "/v1/events/checkout-started",
			eventName: eventmap.CheckoutStartedV1,
			build: func(c echo.Context) (envelopeMessage, []eventmap.FieldError, error) {
				return bindBuild[eventmap.CheckoutStartedRequest](c, func(r eventmap.CheckoutStartedRequest) envelopeMessage { return r.ToProto() })
			},
		},
		{
			path:      "/v1/events/checkout-completed",
			eventName: eventmap.CheckoutCompletedV1,
			build: func(c echo.Context) (envelopeMessage, []eventmap.FieldError, error) {
				return bindBuild[eventmap.CheckoutCompletedRequest](c, func(r eventmap.CheckoutCompletedRequest) envelopeMessage { return r.ToProto() })
			},
		},
		{
			path:      "/v1/events/search-performed",
			eventName: eventmap.SearchPerformedV1,
			build: func(c echo.Context) (envelopeMessage, []eventmap.FieldError, error) {
				return bindBuild[eventmap.SearchPerformedRequest](c, func(r eventmap.SearchPerformedRequest) envelopeMessage { return r.ToProto() })
			},
		},
	}
}

type validator interface {
	Validate() []eventmap.FieldError
}

func bindBuild[T validator](c echo.Context, toProto func(T) envelopeMessage) (envelopeMessage, []eventmap.FieldError, error) {
	var req T
	if err := c.Bind(&req); err != nil {
		return nil, nil, err
	}
	if errs := req.Validate(); len(errs) > 0 {
		return nil, errs, nil
	}
	return toProto(req), nil, nil
}

func handle(pub publisher.Publisher, queueURL, eventName string, build buildFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		msg, fieldErrs, err := build(c)
		if err != nil {
			return c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
		}
		if len(fieldErrs) > 0 {
			return c.JSON(http.StatusBadRequest, fieldErrorsResponse{Errors: fieldErrs})
		}
		wire, err := proto.Marshal(msg)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, errorResponse{Error: err.Error()})
		}
		body := base64.StdEncoding.EncodeToString(wire)
		attrs := map[string]string{
			publisher.AttrEventName: eventName,
			publisher.AttrSchema:    publisher.SchemaValue,
		}
		msgID, err := pub.Publish(c.Request().Context(), body, attrs)
		if err != nil {
			return c.JSON(http.StatusBadGateway, errorResponse{Error: err.Error()})
		}
		return c.JSON(http.StatusAccepted, acceptedResponse{
			EventID:   msg.GetEventId(),
			QueueURL:  queueURL,
			MessageID: msgID,
		})
	}
}
