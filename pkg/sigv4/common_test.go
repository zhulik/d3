package sigv4_test

import (
	"context"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	minioSigner "github.com/minio/minio-go/v7/pkg/signer"
	"github.com/samber/lo"
)

type signRequestFunc func(ctx context.Context, req *http.Request, payloadSum string)
type preSignURLFunc func(ctx context.Context, req *http.Request) string

var (
	awsSigner = v4.NewSigner()

	requestSigners = map[string]signRequestFunc{
		"aws":   signRequestAWS,
		"minio": signRequestMinio,
	}
	urlPreSigners = map[string]preSignURLFunc{
		"aws":   preSignURLAWS,
		"minio": preSignURLMinio,
	}
)

func signRequestAWS(ctx context.Context, req *http.Request, payloadSum string) {
	req.Header.Set("X-Amz-Content-Sha256", payloadSum)

	lo.Must0(
		awsSigner.SignHTTP(ctx, aws.Credentials{
			AccessKeyID:     "test",
			SecretAccessKey: "test",
		}, req, payloadSum, "s3", "local", time.Now()),
	)
}

func signRequestMinio(_ context.Context, req *http.Request, payloadSum string) {
	req.Header.Set("X-Amz-Content-Sha256", payloadSum)

	*req = *minioSigner.SignV4(*req, "test", "test", "", "local")
}

func preSignURLAWS(ctx context.Context, req *http.Request) string {
	qs := req.URL.Query()
	qs.Set("X-Amz-Expires", "120")
	req.URL.RawQuery = qs.Encode()

	url, _ := lo.Must2(
		awsSigner.PresignHTTP(ctx, aws.Credentials{
			AccessKeyID:     "test",
			SecretAccessKey: "test",
		}, req, "UNSIGNED-PAYLOAD", "s3", "local", time.Now()),
	)

	return url
}

func preSignURLMinio(_ context.Context, req *http.Request) string {
	signed := minioSigner.PreSignV4(*req, "test", "test", "", "local", 120)

	return signed.URL.String()
}

type credentialStore struct {
}

func (c *credentialStore) getAccessKeySecret(_ context.Context, _ string) (string, error) {
	return "test", nil
}
