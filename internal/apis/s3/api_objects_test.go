package s3_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zhulik/d3/internal/apictx"
	"github.com/zhulik/d3/internal/apis/s3"
	"github.com/zhulik/d3/internal/apis/s3/middlewares"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/d3/pkg/s3actions"

	"github.com/labstack/echo/v5"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	maxTagCount = 10
	maxTagKey   = 128
	maxTagVal   = 256
)

func TestS3(t *testing.T) {
	t.Parallel()
	RegisterFailHandler(Fail)
	RunSpecs(t, "S3 API Suite")
}

var _ = Describe("ValidateTags", func() {
	When("tags are nil or empty", func() {
		It("returns nil", func() {
			Expect(s3.ValidateTags(nil)).To(Succeed())
			Expect(s3.ValidateTags(map[string]string{})).To(Succeed())
		})
	})

	When("tag count is within limit", func() {
		It("accepts up to 10 tags", func() {
			tags := make(map[string]string, maxTagCount)
			for i := range maxTagCount {
				tags[string(rune('a'+i))] = "v"
			}

			Expect(s3.ValidateTags(tags)).To(Succeed())
		})
	})

	When("more than 10 tags", func() {
		It("returns ErrInvalidTag", func() {
			tags := make(map[string]string, 11)
			for i := range 11 {
				tags[string(rune('a'+i))] = "v"
			}

			err := s3.ValidateTags(tags)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, core.ErrInvalidTag)).To(BeTrue())
		})
	})

	When("tag key exceeds max length", func() {
		It("returns ErrInvalidTag", func() {
			tags := map[string]string{strings.Repeat("k", maxTagKey+1): "v"}
			err := s3.ValidateTags(tags)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, core.ErrInvalidTag)).To(BeTrue())
		})
	})

	When("tag value exceeds max length", func() {
		It("returns ErrInvalidTag", func() {
			tags := map[string]string{"k": strings.Repeat("v", maxTagVal+1)}
			err := s3.ValidateTags(tags)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, core.ErrInvalidTag)).To(BeTrue())
		})
	})

	When("key and value at exact max length", func() {
		It("accepts them", func() {
			tags := map[string]string{
				strings.Repeat("k", maxTagKey): strings.Repeat("v", maxTagVal),
			}
			Expect(s3.ValidateTags(tags)).To(Succeed())
		})
	})
})

var _ = Describe("S3ErrorRenderer", func() {
	When("handler returns ErrUnauthorized", func() {
		It("maps to HTTP 403", func() {
			mw := middlewares.ErrorRenderer()
			handler := mw(func(_ *echo.Context) error {
				return core.ErrUnauthorized
			})

			req := httptest.NewRequest(http.MethodHead, "/bucket/key", nil)
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)

			err := handler(c)
			Expect(err).To(HaveOccurred())

			httpErr := &echo.HTTPError{}
			ok := errors.As(err, &httpErr)
			Expect(ok).To(BeTrue())
			Expect(httpErr.Code).To(Equal(http.StatusForbidden))
		})
	})
})

type capturingAuthorizer struct {
	resource string
}

func (a *capturingAuthorizer) IsAllowed(
	_ context.Context, _ *core.User, _ s3actions.Action, resource string,
) (bool, error) {
	a.resource = resource
	return true, nil
}

var _ = Describe("S3AuthorizerMiddleware", func() {
	When("head object request has no preloaded object", func() {
		It("authorizes against bucket and key from URL", func() {
			captured := &capturingAuthorizer{}
			mw := (&middlewares.Authorizer{Authorizer: captured}).Middleware()

			req := httptest.NewRequest(http.MethodHead, "/my-bucket/path/to/file.txt", nil)
			rec := httptest.NewRecorder()
			c := echo.New().NewContext(req, rec)
			c.SetPath("/:bucket/*")
			c.SetPathValues(echo.PathValues{
				{Name: "bucket", Value: "my-bucket"},
				{Name: "*", Value: "path/to/file.txt"},
			})

			ctx := apictx.Inject(c)
			apiCtx := apictx.FromContext(ctx)
			apiCtx.Action = s3actions.HeadObject
			apiCtx.User = &core.User{Name: "reader"}
			c.SetRequest(c.Request().WithContext(ctx))

			err := mw(func(_ *echo.Context) error { return nil })(c)
			Expect(err).NotTo(HaveOccurred())
			Expect(captured.resource).To(Equal("my-bucket/path/to/file.txt"))
		})
	})
})
