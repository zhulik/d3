package sigv4_test

import (
	"context"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/zhulik/d3/pkg/sigv4"
)

var _ = Describe("Validate", func() {
	When("using a signed request", func() {
		It("validates a valid request", func(ctx context.Context) {
			req := httptest.NewRequest("GET", "/", nil)
			req.Header.Set("x-amz-content-sha256", "UNSIGNED-PAYLOAD")

			signRequest(ctx, req)

			err := sigv4.Validate(ctx, req, &credentialStore{})
			Expect(err).NotTo(HaveOccurred())
		})
	})

	When("using a presigned URL", func() {
		It("validates a valid request", func(ctx context.Context) {
			req := httptest.NewRequest("GET", "/", nil)
			url := preSignURL(ctx, req)

			req = httptest.NewRequest("GET", url, nil)

			err := sigv4.Validate(ctx, req, &credentialStore{})
			Expect(err).NotTo(HaveOccurred())
		})

	})
})
