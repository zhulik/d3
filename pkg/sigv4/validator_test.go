package sigv4_test

import (
	"context"
	"fmt"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/zhulik/d3/pkg/sigv4"
)

var _ = Describe("Validate", func() {
	credentialStore := credentialStore{}

	When("using a signed request", func() {
		It("validates a valid request", func(ctx context.Context) {
			req := httptest.NewRequest("GET", "/foo/bar?baz=qux", nil)
			req.Header.Set("x-amz-content-sha256", "UNSIGNED-PAYLOAD")

			signRequest(ctx, req)

			accessKey, err := sigv4.Validate(ctx, req, credentialStore.getAccessKeySecret)
			Expect(err).NotTo(HaveOccurred())
			Expect(accessKey).To(Equal("test"))
		})
	})

	When("using a presigned URL", func() {
		It("validates a valid request", func(ctx context.Context) {
			req := httptest.NewRequest("GET", "/foo/bar?baz=qux", nil)
			url := preSignURL(ctx, req)

			req = httptest.NewRequest("GET", url, nil)

			accessKey, err := sigv4.Validate(ctx, req, credentialStore.getAccessKeySecret)
			Expect(err).NotTo(HaveOccurred())
			fmt.Println(accessKey)
		})

	})
})
