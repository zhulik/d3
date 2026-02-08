package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/zhulik/d3/internal/core"
)

var ErrUnexpectedStatus = errors.New("unexpected status")

type createUserResponseBody struct {
	Name            string `json:"name"`
	AccessKeyID     string `json:"access_key_id"`
	SecretAccessKey string `json:"secret_access_key"`
}

type Client struct {
	Config     *core.ClientConfig
	signer     *v4.Signer
	httpClient *http.Client
	creds      aws.Credentials
}

func (c *Client) Init(_ context.Context) error {
	c.signer = v4.NewSigner()
	c.httpClient = &http.Client{}
	c.creds = aws.Credentials{
		AccessKeyID:     c.Config.AccessKeyID,
		SecretAccessKey: c.Config.AccessKeySecret,
	}

	return nil
}

func (c *Client) ListUsers(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.Config.ServerURL+"/users", nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.doSignedRequest(ctx, req, http.StatusOK)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var users []string

	err = json.NewDecoder(resp.Body).Decode(&users)

	return users, err
}

func (c *Client) CreateUser(ctx context.Context, name string) (core.User, error) {
	body := map[string]string{"name": name}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return core.User{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Config.ServerURL+"/users", bytes.NewReader(jsonBody))
	if err != nil {
		return core.User{}, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.doSignedRequest(ctx, req, http.StatusOK)
	if err != nil {
		return core.User{}, err
	}

	defer resp.Body.Close()

	var response createUserResponseBody

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return core.User{}, err
	}

	return core.User{
		Name:            response.Name,
		AccessKeyID:     response.AccessKeyID,
		SecretAccessKey: response.SecretAccessKey,
	}, nil
}

func (c *Client) UpdateUser(ctx context.Context, name string) (core.User, error) {
	url := c.Config.ServerURL + "/users/" + name

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, nil)
	if err != nil {
		return core.User{}, err
	}

	resp, err := c.doSignedRequest(ctx, req, http.StatusOK)
	if err != nil {
		return core.User{}, err
	}

	defer resp.Body.Close()

	var response createUserResponseBody

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return core.User{}, err
	}

	return core.User{
		Name:            response.Name,
		AccessKeyID:     response.AccessKeyID,
		SecretAccessKey: response.SecretAccessKey,
	}, nil
}

func (c *Client) DeleteUser(ctx context.Context, name string) error {
	url := c.Config.ServerURL + "/users/" + name

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.doSignedRequest(ctx, req, http.StatusNoContent)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
}

// doSignedRequest signs the provided HTTP request using the client's signer and credentials,
// performs the request using the client's httpClient and verifies the response status code
// matches expectedStatus. On success, it returns the *http.Response (caller must close body).
// On unexpected status it reads up to 1KB of the body, closes it to avoid leaks, and returns an error.
func (c *Client) doSignedRequest(ctx context.Context, req *http.Request, expectedStatus int) (*http.Response, error) {
	if err := c.signer.SignHTTP(ctx, c.creds, req, "", "s3", "local", time.Now()); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != expectedStatus {
		return nil, fmt.Errorf("%w: unexpected status %d", ErrUnexpectedStatus, resp.StatusCode)
	}

	return resp, nil
}
