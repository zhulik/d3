package sigv4_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/zhulik/d3/pkg/sigv4"
)

var _ = Describe("Validate", func() {
	credentialStore := credentialStore{}

	When("using a signed request", func() {
		var entries []any
		for _, payloadSum := range []string{"UNSIGNED-PAYLOAD", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"} {
			for _, method := range []string{http.MethodGet, http.MethodHead} {
				for signRequestName, signFn := range requestSigners {
					for _, url := range []string{
						"http://127.0.0.1:8080/foo/bar?baz=qux",
						"http://127.0.0.1:8080/foo/bar?baz",
						"http://127.0.0.1:8080/foo/bar?baz=",
						"http://127.0.0.1:8080/foo/bar/?baz=",
						"http://127.0.0.1:8080/",
						"http://127.0.0.1:8080/?limit=100",
					} {
						entries = append(entries, Entry(fmt.Sprintf("%s %s %s sum=%s", signRequestName, method, url, payloadSum), method, url, payloadSum, signFn))
					}
				}
			}
		}

		table := slices.Concat([]any{
			func(ctx context.Context, method, url string, payloadSum string, signFn signRequestFunc) {
				req := httptest.NewRequest(method, url, nil)
				req.Header.Set("Host", "127.0.0.1:8080")

				signFn(ctx, req, payloadSum)

				accessKey, err := sigv4.Validate(ctx, req, credentialStore.getAccessKeySecret)
				Expect(err).NotTo(HaveOccurred())
				Expect(accessKey).To(Equal("test"))
			},
		}, entries)

		DescribeTable("validates a valid request", table...)
	})

	When("using a presigned URL", func() {
		When("presigned with AWS SDK", func() {
			for preSignURLName, preSignURL := range urlPreSigners {
				When("presigned with "+preSignURLName, func() {
					DescribeTable("validates a valid pre-signed URL", func(ctx context.Context, rawUrl string) {
						req := httptest.NewRequest(http.MethodGet, rawUrl, nil)
						url := preSignURL(ctx, req)

						req = httptest.NewRequest(http.MethodGet, url, nil)

						accessKey, err := sigv4.Validate(ctx, req, credentialStore.getAccessKeySecret)
						Expect(err).NotTo(HaveOccurred())
						Expect(accessKey).To(Equal("test"))
					},
						Entry("/foo/bar?baz=qux", "/foo/bar?baz=qux"),
					)
				})
			}
		})
	})
})
