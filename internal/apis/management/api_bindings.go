package management

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/core"
)

type createBindingRequestBody struct {
	UserName string `json:"user_name"`
	PolicyID string `json:"policy_id"`
}

type APIBindings struct {
	Backend core.ManagementBackend
	Echo    *Echo
}

func (a APIBindings) Init(_ context.Context) error {
	bindings := a.Echo.Group("/bindings")

	bindings.GET("", a.ListBindings)
	bindings.GET("/user/:userName", a.GetBindingsByUser)
	bindings.GET("/policy/:policyID", a.GetBindingsByPolicy)
	bindings.POST("", a.CreateBinding)
	bindings.DELETE("/user/:userName/policy/:policyID", a.DeleteBinding)

	return nil
}

// ListBindings returns a list of all bindings.
func (a APIBindings) ListBindings(c *echo.Context) error {
	bindings, err := a.Backend.GetBindings(c.Request().Context())
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, bindings)
}

// GetBindingsByUser returns bindings for a specific user.
func (a APIBindings) GetBindingsByUser(c *echo.Context) error {
	userName := c.Param("userName")

	bindings, err := a.Backend.GetBindingsByUser(c.Request().Context(), userName)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, bindings)
}

// GetBindingsByPolicy returns bindings for a specific policy.
func (a APIBindings) GetBindingsByPolicy(c *echo.Context) error {
	policyID := c.Param("policyID")

	bindings, err := a.Backend.GetBindingsByPolicy(c.Request().Context(), policyID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, bindings)
}

// CreateBinding creates a new binding.
func (a APIBindings) CreateBinding(c *echo.Context) error {
	r, err := validateBodyChecksumAndParseJSON[createBindingRequestBody](c)
	if err != nil {
		return err
	}

	binding := &core.PolicyBinding{
		UserName: r.UserName,
		PolicyID: r.PolicyID,
	}

	err = a.Backend.CreateBinding(c.Request().Context(), binding)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusCreated, binding)
}

// DeleteBinding deletes a binding.
func (a APIBindings) DeleteBinding(c *echo.Context) error {
	userName := c.Param("userName")
	policyID := c.Param("policyID")

	err := a.Backend.DeleteBinding(c.Request().Context(), &core.PolicyBinding{
		UserName: userName,
		PolicyID: policyID,
	})
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}
