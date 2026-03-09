package ginkgohelpers

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/onsi/gomega/types"
)

// httpStatusCoder is the interface used by S3/API HTTP errors (e.g. echo.HTTPError).
type httpStatusCoder interface {
	HTTPStatusCode() int
}

// s3HTTPErrorMatcher matches errors that implement HTTPStatusCode() int against an expected status code.
type s3HTTPErrorMatcher struct {
	expectedCode int
}

// Match implements types.GomegaMatcher.
func (m *s3HTTPErrorMatcher) Match(actual any) (bool, error) {
	if actual == nil {
		return false, nil
	}

	errVal, ok := actual.(error)
	if !ok {
		return false, fmt.Errorf("BeS3HttpError matcher expects an error, got %T", actual) //nolint:err113
	}

	var httpErr httpStatusCoder
	if !errors.As(errVal, &httpErr) {
		return false, nil
	}

	return httpErr.HTTPStatusCode() == m.expectedCode, nil
}

// FailureMessage implements types.GomegaMatcher.
func (m *s3HTTPErrorMatcher) FailureMessage(actual any) string {
	if actual == nil {
		return "Expected an error with HTTP status code " + strconv.Itoa(m.expectedCode) + ", but got nil"
	}

	errVal, ok := actual.(error)
	if !ok {
		return fmt.Sprintf("Expected an error, got %T", actual)
	}

	var httpErr httpStatusCoder
	if !errors.As(errVal, &httpErr) {
		return "Expected error to have HTTPStatusCode(), but it does not: " + errVal.Error()
	}

	got := httpErr.HTTPStatusCode()

	return fmt.Sprintf("Expected HTTP status code %d, but got %d", m.expectedCode, got)
}

// NegatedFailureMessage implements types.GomegaMatcher.
func (m *s3HTTPErrorMatcher) NegatedFailureMessage(actual any) string {
	if actual == nil {
		return "Expected not to get HTTP status " + strconv.Itoa(m.expectedCode) + ", but got nil error"
	}

	errVal, ok := actual.(error)
	if !ok {
		return fmt.Sprintf("Expected an error, got %T", actual)
	}

	var httpErr httpStatusCoder
	if !errors.As(errVal, &httpErr) {
		return "Expected error not to have HTTP status " + strconv.Itoa(m.expectedCode) + ", but error has no HTTPStatusCode()" //nolint:lll
	}

	return fmt.Sprintf("Expected HTTP status code not to be %d, but got %d", m.expectedCode, httpErr.HTTPStatusCode())
}

// BeS3HttpError returns a Gomega matcher that asserts the actual value is an error
// whose error chain contains a type with HTTPStatusCode() int equal to expectedCode.
// Use with: Expect(err).To(BeS3HttpError(304)).
func BeS3HttpError(expectedCode int) types.GomegaMatcher {
	return &s3HTTPErrorMatcher{expectedCode: expectedCode}
}
