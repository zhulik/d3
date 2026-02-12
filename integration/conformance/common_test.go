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

// runApp starts the app and returns port, cancel, admin credentials, temp dir, and ManagementBackend.
func runApp(ctx context.Context) (int, context.CancelFunc, string, core.ManagementBackend) {
	appCtx, cancelApp := context.WithCancel(ctx)

	tempDir := lo.Must(os.MkdirTemp("/tmp", "d3-conformance-"))

	appConfig := &core.Config{
		Environment:               "test",
		StorageBackend:            core.StorageBackendFolder,
		FolderStorageBackendPath:  tempDir,
		ManagementBackend:         core.ManagementBackendYAML,
		ManagementBackendYAMLPath: filepath.Join(tempDir, "management.yaml"),
		ManagementBackendTmpPath:  tempDir,
		RedisAddress:              "localhost:6379",
		Port:                      randomPort(),
		HealthCheckPort:           randomPort(),
	}

	app := application.NewServer(appConfig)
	lo.Must0(app.Init(ctx))

	time.Sleep(100 * time.Millisecond)

	mgmtBackend := pal.MustInvoke[core.ManagementBackend](ctx, app)

	go func() {
		lo.Must0(app.Run(appCtx))
	}()

	time.Sleep(100 * time.Millisecond)

	return appConfig.Port, cancelApp, tempDir, mgmtBackend
}

func createS3Client(ctx context.Context, port int, accessKeyID, secretAccessKey string) *s3.Client {
	cfg := lo.Must(config.LoadDefaultConfig(ctx,
		config.WithBaseEndpoint(fmt.Sprintf("http://localhost:%d", port)),
		config.WithRegion("local"),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "test"),
		),
	))

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.RetryMaxAttempts = 1
	})
}

func prepareConformanceTests(ctx context.Context) (*s3.Client, string, int, context.CancelFunc, string, core.ManagementBackend) {
	port, cancelApp, tempDir, mgmtBackend := runApp(context.Background()) //nolint:contextcheck

	admin := lo.Must(mgmtBackend.GetUserByName(ctx, "admin"))

	bucketName := "conformance-bucket-" + uuid.NewString()

	s3Client := createS3Client(ctx, port, admin.AccessKeyID, admin.SecretAccessKey)

	lo.Must(s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: &bucketName,
	}))

	return s3Client, bucketName, port, cancelApp, tempDir, mgmtBackend
}

func cleanupS3(ctx context.Context, cancelApp context.CancelFunc, s3Client *s3.Client, bucketName string, tempDir string) {
	objects := lo.Must(s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: &bucketName,
	}))
	if len(objects.Contents) > 0 {
		lo.Must(s3Client.DeleteObjects(ctx, &s3.DeleteObjectsInput{
			Bucket: &bucketName,
			Delete: &types.Delete{
				Objects: lo.Map(objects.Contents, func(object types.Object, _ int) types.ObjectIdentifier {
					return types.ObjectIdentifier{Key: object.Key}
				}),
			},
		}))
	}

	lo.Must(s3Client.DeleteBucket(ctx, &s3.DeleteBucketInput{
		Bucket: &bucketName,
	}))

	cancelApp()
	lo.Must0(os.RemoveAll(tempDir))
}
