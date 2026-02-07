package management

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/credentials"
)

type createUserRequestBody struct {
	Name string `json:"name"`
}

type createUserResponseBody struct {
	Name            string `json:"name"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}

type APIUsers struct {
	Backend core.ManagementBackend
	Echo    *Echo
}

func (a APIUsers) Init(_ context.Context) error {
	users := a.Echo.Group("/users")

	users.GET("", a.ListUsers)
	users.POST("", a.CreateUser)
	users.PUT("/:userName", a.UpdateUser)
	users.DELETE("/:userName", a.DeleteUser)

	return nil
}

// ListUsers returns a list of users.
func (a APIUsers) ListUsers(c *echo.Context) error {
	users, err := a.Backend.GetUsers(c.Request().Context())
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, users)
}

// CreateUser creates a user.
func (a APIUsers) CreateUser(c *echo.Context) error {
	r := createUserRequestBody{}
	if err := c.Bind(&r); err != nil {
		return err
	}

	accessKeyID, secretAccessKey := credentials.GenerateCredentials()
	user := core.User{
		Name:            r.Name,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
	}

	err := a.Backend.CreateUser(c.Request().Context(), user)
	if err != nil {
		return err
	}

	response := createUserResponseBody{
		Name:            user.Name,
		AccessKeyID:     user.AccessKeyID,
		SecretAccessKey: user.SecretAccessKey,
	}

	return c.JSON(http.StatusOK, response)
}

// UpdateUser updates the user.
func (a APIUsers) UpdateUser(c *echo.Context) error {
	userName := c.Param("userName")

	accessKeyID, secretAccessKey := credentials.GenerateCredentials()
	user := core.User{
		Name:            userName,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
	}

	err := a.Backend.UpdateUser(c.Request().Context(), user)
	if err != nil {
		return err
	}

	response := createUserResponseBody{
		Name:            user.Name,
		AccessKeyID:     user.AccessKeyID,
		SecretAccessKey: user.SecretAccessKey,
	}

	return c.JSON(http.StatusOK, response)
}

// DeleteUser deletes the user.
func (a APIUsers) DeleteUser(c *echo.Context) error {
	userName := c.Param("userName")

	err := a.Backend.DeleteUser(c.Request().Context(), userName)
	if err != nil {
		return err
	}

	return c.NoContent(http.StatusNoContent)
}
