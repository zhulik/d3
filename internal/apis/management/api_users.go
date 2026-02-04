package management

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v5"
)

type APIUsers struct {
	Echo *Echo
}

func (a APIUsers) Init(_ context.Context) error {
	users := a.Echo.Group("/users")

	users.GET("", a.ListUsers)
	users.POST("", a.CreateUser)
	users.PUT("/:userID", a.UpdateUser)
	users.DELETE("/:userID", a.DeleteUser)

	return nil
}

// ListUsers returns a list of users.
func (a APIUsers) ListUsers(c *echo.Context) error {
	return c.JSON(http.StatusOK, []string{})
}

// CreateUser creates a user.
func (a APIUsers) CreateUser(c *echo.Context) error {
	return c.JSON(http.StatusOK, []string{})
}

// UpdateUser updates the user.
func (a APIUsers) UpdateUser(c *echo.Context) error {
	return c.JSON(http.StatusOK, []string{})
}

// DeleteUser updates the user.
func (a APIUsers) DeleteUser(c *echo.Context) error {
	return c.JSON(http.StatusOK, []string{})
}
