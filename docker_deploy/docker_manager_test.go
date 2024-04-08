package docker_deploy

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"go.viam.com/rdk/logging"
)

var imageName = "mcr.microsoft.com/dotnet/samples"
var repoDigest = "sha256:d41fe80991d7c26ad43b052bb87c68a216a365c143623a62b5a5963fcdb77eb1"

var options = map[string]interface{}{
	"Hostname":   "my-container",
	"User":       "root",
	"Image":      "alpine",
	"WorkingDir": "/app",
	"StopSignal": "SIGTERM",
}

var hostOptions = map[string]interface{}{
	"NetworkMode": "bridge",
	"AutoRemove":  "true",
	"Privileged":  "false",
	"ShmSize":     "67108864", // 64MB in bytes
}

func docker_manager_test_setup(t *testing.T) (logging.Logger, DockerManager) {
	logger := logging.NewTestLogger(t)
	dm, err := NewLocalDockerManager(logger)
	assert.NoError(t, err)

	t.Cleanup(func() {
		cli, err := client.NewClientWithOpts(client.FromEnv)
		if err != nil {
			t.Error(err)
		}
		containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
		assert.NoError(t, err)
		for _, c := range containers {
			cli.ContainerRemove(context.Background(), c.ID, container.RemoveOptions{Force: true})
		}
		if imageExists, _ := dm.ImageExists(repoDigest); imageExists {
			err = dm.RemoveImageByRepoDigest(repoDigest)
			assert.NoError(t, err)
		}
	})

	return logger, dm
}

func TestListContainers(t *testing.T) {
	_, dm := docker_manager_test_setup(t)
	containers, err := dm.ListContainers()
	if err != nil {
		t.Fatal(err)
	}

	for _, container := range containers {
		t.Logf("%#v", container)
	}
}

func TestListImages(t *testing.T) {
	_, dm := docker_manager_test_setup(t)

	images, err := dm.ListImages()
	if err != nil {
		t.Fatal(err)
	}

	for _, image := range images {
		t.Logf("%#v", image)
	}
}

func TestGetContainerImageDigest(t *testing.T) {
	logger, dm := docker_manager_test_setup(t)
	ctx, _ := context.WithCancel(context.Background())
	err := dm.PullImage(ctx, imageName, repoDigest)
	assert.NoError(t, err)

	container, err := dm.CreateContainer(imageName, repoDigest, []string{"sleep", "1000"}, options, hostOptions, logger, ctx)
	assert.NoError(t, err)
	digest, err := dm.GetContainerImageDigest(container.GetContainerId())
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "sha256:d41fe80991d7c26ad43b052bb87c68a216a365c143623a62b5a5963fcdb77eb1", digest)
}

// TODO: This test will fail until we start the container. We need to add more setup code.
func TestGetContainersRunningImage(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())

	logger, dm := docker_manager_test_setup(t)
	err := dm.PullImage(ctx, imageName, repoDigest)
	assert.NoError(t, err)

	container, err := dm.CreateContainer(imageName, repoDigest, []string{"sleep", "1000"}, options, hostOptions, logger, ctx)
	assert.NoError(t, err)

	err = dm.StartContainer(container.GetContainerId())
	assert.NoError(t, err)

	containers, err := dm.GetContainersRunningImage("sha256:d41fe80991d7c26ad43b052bb87c68a216a365c143623a62b5a5963fcdb77eb1")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1, len(containers))
	for _, container := range containers {
		t.Logf("%#v", container)
	}
}

func TestImageExists(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())

	_, dm := docker_manager_test_setup(t)

	exists, err := dm.ImageExists(repoDigest)
	assert.NoError(t, err)
	assert.False(t, exists)

	err = dm.PullImage(ctx, imageName, repoDigest)
	assert.NoError(t, err)
	exists, err = dm.ImageExists(repoDigest)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestImagePull(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())

	_, dm := docker_manager_test_setup(t)

	exists, err := dm.ImageExists(repoDigest)
	assert.NoError(t, err)
	assert.False(t, exists)

	err = dm.PullImage(ctx, imageName, repoDigest)
	assert.NoError(t, err)
	exists, err = dm.ImageExists(repoDigest)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestImageRemove(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())

	_, dm := docker_manager_test_setup(t)

	exists, err := dm.ImageExists(repoDigest)
	assert.NoError(t, err)
	assert.False(t, exists)

	err = dm.PullImage(ctx, imageName, repoDigest)
	assert.NoError(t, err)
	exists, err = dm.ImageExists(repoDigest)
	assert.NoError(t, err)
	assert.True(t, exists)

	err = dm.RemoveImageByRepoDigest(repoDigest)
	assert.NoError(t, err)

	exists, err = dm.ImageExists(repoDigest)
	assert.NoError(t, err)
	assert.False(t, exists)
}
