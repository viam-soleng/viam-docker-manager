package docker

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.viam.com/rdk/logging"
)

func TestImageExists(t *testing.T) {
	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	logger := logging.NewTestLogger(t)

	image := NewDockerComposeImage("mcr.microsoft.com/dotnet/samples", "sha256:test", "", logger, cancelCtx, cancelFunc)
	assert.False(t, image.Exists())

	image = NewDockerComposeImage("mcr.microsoft.com/dotnet/samples", "sha256:d41fe80991d7c26ad43b052bb87c68a216a365c143623a62b5a5963fcdb77eb1", "", logger, cancelCtx, cancelFunc)
	assert.True(t, image.Exists())
}

func TestImagePull(t *testing.T) {
	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	logger := logging.NewTestLogger(t)
	manager := NewLocalDockerManager(logger)

	image := NewDockerComposeImage("mcr.microsoft.com/dotnet/samples", "sha256:d41fe80991d7c26ad43b052bb87c68a216a365c143623a62b5a5963fcdb77eb1", "", logger, cancelCtx, cancelFunc)
	if image.Exists() {
		assert.NoError(t, image.Remove())
	}

	assert.False(t, image.Exists())

	assert.NoError(t, manager.PullImage("ubuntu", "sha256:d41fe80991d7c26ad43b052bb87c68a216a365c143623a62b5a5963fcdb77eb1"), "Image should be pulled")

	imageId := image.GetImageId()
	assert.NotEmpty(t, imageId)
	logger.Infof("ImageID: %v", imageId)

	assert.True(t, image.Exists())
}

func TestImageRemove(t *testing.T) {
	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	logger := logging.NewTestLogger(t)
	manager := NewLocalDockerManager(logger)

	image := NewDockerComposeImage("mcr.microsoft.com/dotnet/samples", "sha256:d41fe80991d7c26ad43b052bb87c68a216a365c143623a62b5a5963fcdb77eb1", "", logger, cancelCtx, cancelFunc)
	if image.Exists() {
		assert.NoError(t, image.Remove())
	}

	assert.False(t, image.Exists())

	assert.NoError(t, manager.PullImage("ubuntu", "sha256:d41fe80991d7c26ad43b052bb87c68a216a365c143623a62b5a5963fcdb77eb1"), "Image should be pulled")

	imageId := image.GetImageId()
	assert.NotEmpty(t, imageId)
	logger.Infof("ImageID: %v", imageId)

	assert.True(t, image.Exists())

	assert.NoError(t, image.Remove())

	assert.False(t, image.Exists())
}

func TestGetImageId(t *testing.T) {
	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	logger := logging.NewTestLogger(t)

	image := NewDockerComposeImage("mcr.microsoft.com/dotnet/samples", "sha256:d41fe80991d7c26ad43b052bb87c68a216a365c143623a62b5a5963fcdb77eb1", "", logger, cancelCtx, cancelFunc)
	assert.True(t, image.Exists())

	imageId := image.GetImageId()
	assert.NotEmpty(t, imageId)
	logger.Infof("ImageID: %v", imageId)
}

func TestIsRunning(t *testing.T) {
	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	logger := logging.NewTestLogger(t)
	manager := NewLocalDockerManager(logger)

	image := NewDockerComposeImage("ubuntu", "sha256:2b7412e6465c3c7fc5bb21d3e6f1917c167358449fecac8176c6e496e5c1f05f", "", logger, cancelCtx, cancelFunc)
	if !image.Exists() {
		assert.NoError(t, manager.PullImage("ubuntu", "sha256:2b7412e6465c3c7fc5bb21d3e6f1917c167358449fecac8176c6e496e5c1f05f"), "Image should be pulled")
	}

	assert.True(t, image.Exists(), "Image should exist")

	isRunning, err := image.IsRunning()
	assert.NoError(t, err, "Error should be nil")
	assert.False(t, isRunning, "Image should not be running")

	assert.NoError(t, image.Start(), "Image should be started")

	isRunning, err = image.IsRunning()
	assert.NoError(t, err, "Error should be nil")
	assert.True(t, isRunning, "Image should be running")

	assert.NoError(t, image.Stop(), "Image should be stopped")

	isRunning, err = image.IsRunning()
	assert.NoError(t, err, "Error should be nil")
	assert.False(t, isRunning, "Image should not be running")
}
