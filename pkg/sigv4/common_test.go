package sigv4_test

import (
	"context"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/samber/lo"
)

var (
	signer = v4.NewSigner() //nolint:gochecknoglobals
)

func signRequest(ctx context.Context, req *http.Request) {
	lo.Must0(
		signer.SignHTTP(ctx, aws.Credentials{
			AccessKeyID:     "test",
			SecretAccessKey: "test",
		}, req, "UNSIGNED-PAYLOAD", "s3", "local", time.Now()),
	)
}

func preSignURL(ctx context.Context, req *http.Request) string {
	qs := req.URL.Query()
	qs.Set("X-Amz-Expires", "120")
	req.URL.RawQuery = qs.Encode()

	url, _ := lo.Must2(
		signer.PresignHTTP(ctx, aws.Credentials{
			AccessKeyID:     "test",
			SecretAccessKey: "test",
		}, req, "UNSIGNED-PAYLOAD", "s3", "local", time.Now()),
	)

	return url
}

type credentialStore struct {
}

func (c *credentialStore) getAccessKeySecret(_ context.Context, _ string) (string, error) {
	return "test", nil
}
