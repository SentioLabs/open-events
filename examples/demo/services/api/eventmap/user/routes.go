package user

import (
	"github.com/labstack/echo/v4"

	"github.com/sentiolabs/open-events/examples/demo/services/api/eventmap"
)

// Routes returns all user-domain event routes.
func Routes() []eventmap.Route {
	return []eventmap.Route{
		{Path: "/v1/events/user/auth/signup", EventName: AuthSignupV1, Build: buildAuthSignup},
		{Path: "/v1/events/user/auth/login", EventName: AuthLoginV1, Build: buildAuthLogin},
		{Path: "/v1/events/user/auth/logout", EventName: AuthLogoutV1, Build: buildAuthLogout},
		{Path: "/v1/events/user/cart/checkout", EventName: CartCheckoutV1, Build: buildCartCheckout},
		{Path: "/v1/events/user/cart/purchase", EventName: CartPurchaseV1, Build: buildCartPurchase},
		{Path: "/v1/events/user/cart/item_added", EventName: CartItemAddedV1, Build: buildCartItemAdded},
	}
}

func buildAuthSignup(c echo.Context) (eventmap.EnvelopeMessage, []eventmap.FieldError, error) {
	return eventmap.BindBuild[AuthSignupRequest](c, func(r AuthSignupRequest) eventmap.EnvelopeMessage { return r.ToProto() })
}

func buildAuthLogin(c echo.Context) (eventmap.EnvelopeMessage, []eventmap.FieldError, error) {
	return eventmap.BindBuild[AuthLoginRequest](c, func(r AuthLoginRequest) eventmap.EnvelopeMessage { return r.ToProto() })
}

func buildAuthLogout(c echo.Context) (eventmap.EnvelopeMessage, []eventmap.FieldError, error) {
	return eventmap.BindBuild[AuthLogoutRequest](c, func(r AuthLogoutRequest) eventmap.EnvelopeMessage { return r.ToProto() })
}

func buildCartCheckout(c echo.Context) (eventmap.EnvelopeMessage, []eventmap.FieldError, error) {
	return eventmap.BindBuild[CartCheckoutRequest](c, func(r CartCheckoutRequest) eventmap.EnvelopeMessage { return r.ToProto() })
}

func buildCartPurchase(c echo.Context) (eventmap.EnvelopeMessage, []eventmap.FieldError, error) {
	return eventmap.BindBuild[CartPurchaseRequest](c, func(r CartPurchaseRequest) eventmap.EnvelopeMessage { return r.ToProto() })
}

func buildCartItemAdded(c echo.Context) (eventmap.EnvelopeMessage, []eventmap.FieldError, error) {
	return eventmap.BindBuild[CartItemAddedRequest](c, func(r CartItemAddedRequest) eventmap.EnvelopeMessage { return r.ToProto() })
}
