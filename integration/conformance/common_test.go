package conformance_test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/zhulik/d3/internal/application"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/pal"
)

func randomPort() int {
	return 10000 + rand.Intn(10000)
}

func prepareS3(ctx context.Context, port int, adminAccessKeyID, adminSecretAccessKey string) (*s3.Client, *string) {
	bucketName := lo.ToPtr("conformance-bucket-" + uuid.NewString())

	cfg := lo.Must(config.LoadDefaultConfig(ctx,
		config.WithBaseEndpoint(fmt.Sprintf("http://localhost:%d", port)),
		config.WithRegion("local"),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(adminAccessKeyID, adminSecretAccessKey, "test"),
		),
	))

	s3Client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.RetryMaxAttempts = 1
	})

	lo.Must(s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: bucketName,
	}))

	return s3Client, bucketName
}

func runApp(ctx context.Context) (int, context.CancelFunc, string, string, string) {
	appCtx, cancelApp := context.WithCancel(ctx)

	tempDir := lo.Must(os.MkdirTemp("/tmp", "d3-conformance-"))

	appConfig := &core.Config{
		Environment:               "test",
		StorageBackend:            core.StorageBackendFolder,
		FolderStorageBackendPath:  tempDir,
		ManagementBackend:         core.ManagementBackendYAML,
		ManagementBackendYAMLPath: filepath.Join(tempDir, "management.yaml"),
		RedisAddress:              "localhost:6379",
		Port:                      randomPort(),
		HealthCheckPort:           randomPort(),
	}

	app := application.NewServer(appConfig)
	lo.Must0(app.Init(ctx))

	time.Sleep(100 * time.Millisecond)

	userRepository := pal.MustInvoke[core.ManagementBackend](ctx, app)
	adminAccessKeyID, adminSecretAccessKey := userRepository.AdminCredentials()

	go func() {
		lo.Must0(app.Run(appCtx))
	}()

	time.Sleep(100 * time.Millisecond)

	return appConfig.Port, cancelApp, adminAccessKeyID, adminSecretAccessKey, tempDir
}

func prepareConformanceTests(ctx context.Context) (*s3.Client, *string, context.CancelFunc, string) {
	port, cancelApp, adminAccessKeyID, adminSecretAccessKey, tempDir := runApp(context.Background()) //nolint:contextcheck

	s3Client, bucketName := prepareS3(ctx, port, adminAccessKeyID, adminSecretAccessKey)

	return s3Client, bucketName, cancelApp, tempDir
}

func cleanupS3(ctx context.Context, s3Client *s3.Client, bucketName *string, tempDir string) {
	objects := lo.Must(s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: bucketName,
	}))
	if len(objects.Contents) > 0 {
		lo.Must(s3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: bucketName,
			Delete: &types.Delete{
				Objects: lo.Map(objects.Contents, func(object types.Object, _ int) types.ObjectIdentifier {
					return types.ObjectIdentifier{Key: object.Key}
				}),
			},
		}))
	}

	lo.Must(s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: bucketName,
	}))

	lo.Must0(os.RemoveAll(tempDir))
}
