package management

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/iampol"
	"github.com/zhulik/d3/pkg/smartio"
)

type APIPolicies struct {
	Backend core.ManagementBackend
	Echo    *Echo
}

func (a APIPolicies) Init(_ context.Context) error {
	policies := a.Echo.Group("/policies")

	policies.GET("", a.ListPolicies)
	policies.GET("/:policyID", a.GetPolicy)
	policies.POST("", a.CreatePolicy)
	policies.PUT("/:policyID", a.UpdatePolicy)
	policies.DELETE("/:policyID", a.DeletePolicy)

	return nil
}

// ListPolicies returns a list of policy IDs.
func (a APIPolicies) ListPolicies(c *echo.Context) error {
	policies, err := a.Backend.GetPolicies(c.Request().Context())
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, policies)
}

// CreatePolicy creates a policy.
func (a APIPolicies) CreatePolicy(c *echo.Context) error {
	raw, _, err := smartio.ReadAllAndHash(c.Request().Body)
	if err != nil {
		return err
	}

	policy, err := iampol.Parse(raw)
	if err != nil {
		return err
	}

	err = a.Backend.CreatePolicy(c.Request().Context(), policy)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusCreated)
}

// UpdatePolicy updates the policy.
func (a APIPolicies) UpdatePolicy(c *echo.Context) error {
	policyID := c.Param("policyID")

	raw, _, err := smartio.ReadAllAndHash(c.Request().Body)
	if err != nil {
		return err
	}

	policy, err := iampol.Parse(raw)
	if err != nil {
		return err
	}

	policy.ID = policyID

	err = a.Backend.UpdatePolicy(c.Request().Context(), policy)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusOK)
}

// DeletePolicy deletes the policy.
func (a APIPolicies) DeletePolicy(c *echo.Context) error {
	policyID := c.Param("policyID")

	err := a.Backend.DeletePolicy(c.Request().Context(), policyID)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}

// GetPolicy retrieves a policy by its ID.
func (a APIPolicies) GetPolicy(c *echo.Context) error {
	policyID := c.Param("policyID")

	policy, err := a.Backend.GetPolicyByID(c.Request().Context(), policyID)
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, policy)
}
