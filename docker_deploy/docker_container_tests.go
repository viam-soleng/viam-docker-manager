package docker_deploy

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.viam.com/rdk/logging"
)

func TestGetImageId(t *testing.T) {
	cancelCtx, _ := context.WithCancel(context.Background())
	logger := logging.NewTestLogger(t)
	dm, err := NewLocalDockerManager(logger)
	assert.NoError(t, err)

	container, err := dm.CreateContainer("mcr.microsoft.com/dotnet/samples", "sha256:d41fe80991d7c26ad43b052bb87c68a216a365c143623a62b5a5963fcdb77eb1", []string{}, []string{}, logger, cancelCtx)
	assert.NoError(t, err, "Error should be nil")

	imageId, err := container.GetImageId()
	assert.NoError(t, err, "Error should be nil")
	assert.NotEmpty(t, imageId)
	logger.Infof("ImageID: %v", imageId)
}

func TestIsRunning(t *testing.T) {
	cancelCtx, _ := context.WithCancel(context.Background())
	logger := logging.NewTestLogger(t)
	dm, err := NewLocalDockerManager(logger)
	assert.NoError(t, err)

	container, err := dm.CreateContainer("ubuntu", "sha256:2b7412e6465c3c7fc5bb21d3e6f1917c167358449fecac8176c6e496e5c1f05f", []string{}, []string{}, logger, cancelCtx)
	assert.NoError(t, err, "Error should be nil")

	isRunning, err := container.IsRunning()
	assert.NoError(t, err, "Error should be nil")
	assert.False(t, isRunning, "Image should not be running")

	assert.NoError(t, dm.StartContainer(container.GetContainerId()), "Image should be started")

	isRunning, err = container.IsRunning()
	assert.NoError(t, err, "Error should be nil")
	assert.True(t, isRunning, "Image should be running")

	assert.NoError(t, dm.StopContainer(container.GetContainerId()), "Image should be stopped")

	isRunning, err = container.IsRunning()
	assert.NoError(t, err, "Error should be nil")
	assert.False(t, isRunning, "Image should not be running")
}
