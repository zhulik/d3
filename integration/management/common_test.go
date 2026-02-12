package management_test

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/samber/lo"
	"github.com/zhulik/d3/internal/application"
	"github.com/zhulik/d3/internal/client/apiclient"
	"github.com/zhulik/d3/internal/core"
	"github.com/zhulik/pal"
)

func randomPort() int {
	return 10000 + rand.Intn(10000)
}

func runApp(ctx context.Context) (int, context.CancelFunc, string, string, string) {
	appCtx, cancelApp := context.WithCancel(ctx)

	tempDir := lo.Must(os.MkdirTemp("/tmp", "d3-management-"))

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

	app := application.NewServer(appConfig)
	lo.Must0(app.Init(ctx))

	time.Sleep(100 * time.Millisecond)

	userRepository := pal.MustInvoke[core.ManagementBackend](ctx, app)
	admin := lo.Must(userRepository.GetUserByName(ctx, "admin"))

	go func() {
		lo.Must0(app.Run(appCtx))
	}()

	time.Sleep(100 * time.Millisecond)

	return appConfig.ManagementPort, cancelApp, admin.AccessKeyID, admin.SecretAccessKey, tempDir
}

func prepareManagementTests(ctx context.Context) (*apiclient.Client, context.CancelFunc, string) {
	managementPort, cancelApp, adminAccessKeyID, adminSecretAccessKey, tempDir := runApp(context.Background()) //nolint:contextcheck

	clientConfig := &core.ClientConfig{
		ServerURL:       fmt.Sprintf("http://localhost:%d", managementPort),
		AccessKeyID:     adminAccessKeyID,
		AccessKeySecret: adminSecretAccessKey,
	}

	client := &apiclient.Client{
		Config: clientConfig,
	}
	lo.Must0(client.Init(ctx))

	return client, cancelApp, tempDir
}

func cleanupManagementTests(_ context.Context, tempDir string) {
	lo.Must0(os.RemoveAll(tempDir))
}
