package testhelpers

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
	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	minioCreds "github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/samber/lo"
	"github.com/zhulik/d3/internal/application"
	"github.com/zhulik/d3/internal/client/apiclient"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/pal"
)

type App struct {
	cancelApp      context.CancelFunc
	pal            *pal.Pal
	s3Port         int
	managementPort int
	tempDir        string
	bucketName     string
}

func NewApp() *App {
	ctx, cancelApp := context.WithCancel(context.Background())

	tempDir := lo.Must(os.MkdirTemp("/tmp", "d3-"))

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
		ManagementPort:            randomPort(),
	}

	pal := application.NewServer(appConfig)
	lo.Must0(pal.Init(ctx))

	go func() {
		lo.Must0(pal.Run(ctx))
	}()

	time.Sleep(100 * time.Millisecond)

	app := &App{
		cancelApp:      cancelApp,
		pal:            pal,
		s3Port:         appConfig.Port,
		managementPort: appConfig.ManagementPort,
		tempDir:        tempDir,
		bucketName:     "bucket-" + uuid.NewString(),
	}

	s3Client := app.S3Client(ctx, "admin")

	lo.Must(s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: lo.ToPtr(app.bucketName),
	}))

	return app
}

func (a *App) Stop(_ context.Context) {
	a.cancelApp()
	lo.Must0(os.RemoveAll(a.tempDir))
}

func (a *App) ManagementClient(ctx context.Context) *apiclient.Client {
	managementBackend := pal.MustInvoke[core.ManagementBackend](ctx, a.pal)
	admin := lo.Must(managementBackend.GetUserByName(ctx, "admin"))

	clientConfig := &core.ClientConfig{
		ServerURL:       fmt.Sprintf("http://localhost:%d", a.managementPort),
		AccessKeyID:     admin.AccessKeyID,
		AccessKeySecret: admin.SecretAccessKey,
	}

	client := &apiclient.Client{
		Config: clientConfig,
	}
	lo.Must0(client.Init(ctx))

	return client
}

func (a *App) S3Client(ctx context.Context, username string) *s3.Client {
	managementBackend := pal.MustInvoke[core.ManagementBackend](ctx, a.pal)
	user := lo.Must(managementBackend.GetUserByName(ctx, username))

	cfg := lo.Must(config.LoadDefaultConfig(ctx,
		config.WithBaseEndpoint(fmt.Sprintf("http://localhost:%d", a.s3Port)),
		config.WithRegion("local"),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(user.AccessKeyID, user.SecretAccessKey, ""),
		),
	))

	return s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = true
		o.RetryMaxAttempts = 1
	})
}

func (a *App) MinioClient(ctx context.Context, username string) *minio.Client {
	managementBackend := pal.MustInvoke[core.ManagementBackend](ctx, a.pal)
	user := lo.Must(managementBackend.GetUserByName(ctx, username))

	endpoint := fmt.Sprintf("localhost:%d", a.s3Port)
	client := lo.Must(minio.New(endpoint, &minio.Options{
		Creds:        minioCreds.NewStaticV4(user.AccessKeyID, user.SecretAccessKey, ""),
		Secure:       false, // no TLS
		Region:       "local",
		BucketLookup: minio.BucketLookupPath,
		MaxRetries:   1,
	}))

	return client
}

func (a *App) BucketName() string {
	return a.bucketName
}

func (a *App) ManagementBackend(ctx context.Context) core.ManagementBackend {
	return pal.MustInvoke[core.ManagementBackend](ctx, a.pal)
}

func randomPort() int {
	return 10000 + rand.Intn(10000) //nolint:gosec
}
