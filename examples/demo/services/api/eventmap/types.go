package eventmap

import (
	"github.com/labstack/echo/v4"
	"google.golang.org/protobuf/proto"
)

// FieldError describes a single validation failure.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// EnvelopeMessage is the subset of a generated proto envelope that the handler
// relies on for publishing and response building.
type EnvelopeMessage interface {
	proto.Message
	GetEventId() string
}

// BuildFunc decodes the request body, validates it, and returns the proto
// message ready to publish. If fieldErrs is non-empty, the handler returns 400.
type BuildFunc func(c echo.Context) (msg EnvelopeMessage, fieldErrs []FieldError, err error)

// Route pairs an HTTP path with an event name and a BuildFunc.
type Route struct {
	Path      string
	EventName string
	Build     BuildFunc
}

// Validator is implemented by per-action Request structs.
type Validator interface {
	Validate() []FieldError
}

// BindBuild binds the JSON body into a T, validates it, and calls toProto.
// It is the generic helper shared by all domain route builders.
func BindBuild[T Validator](c echo.Context, toProto func(T) EnvelopeMessage) (EnvelopeMessage, []FieldError, error) {
	var req T
	if err := c.Bind(&req); err != nil {
		return nil, nil, err
	}
	if errs := req.Validate(); len(errs) > 0 {
		return nil, errs, nil
	}
	return toProto(req), nil, nil
}
