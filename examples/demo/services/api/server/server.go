package server

import (
	"encoding/base64"
	"net/http"

	"github.com/labstack/echo/v4"
	"google.golang.org/protobuf/proto"

	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/device"
	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap/user"
	"github.com/sentiolabs/open-events/examples/demo/services/api/publisher"
)

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

	routes := append(user.Routes(), device.Routes()...)
	for _, r := range routes {
		e.POST(r.Path, handle(pub, queueURL, r.EventName, r.Build))
	}

	return e
}

func handle(pub publisher.Publisher, queueURL, eventName string, build eventmap.BuildFunc) echo.HandlerFunc {
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
