package docker_deploy

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
)

func setupDependencies(t *testing.T) (resource.Config, resource.Dependencies) {
	// Gotta set the environment variable because the running code expects it to be there
	os.Setenv("VIAM_MODULE_DATA", os.TempDir())
	cfg := resource.Config{
		Name:  "movementsensor",
		Model: Model,
		API:   sensor.API,
		ConvertedAttributes: &Config{
			ImageName:  "ubuntu",
			RepoDigest: "sha256:218bb51abbd1864df8be26166f847547b3851a89999ca7bfceb85ca9b5d2e95d",
			ComposeFile: []string{
				"services:",
				"  app:",
				"    image: ubuntu@sha256:218bb51abbd1864df8be26166f847547b3851a89999ca7bfceb85ca9b5d2e95d",
				"    command: sleep 10",
				"    working_dir: /root",
			},
		},
	}

	t.Cleanup(func() {
		cli, err := client.NewClientWithOpts(client.FromEnv)
		if err != nil {
			t.Error(err)
		}
		containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
		assert.NoError(t, err)
		for _, container := range containers {
			cli.ContainerRemove(context.Background(), container.ID, types.ContainerRemoveOptions{Force: true})
		}
	})

	return cfg, resource.Dependencies{}
}

func TestReconfigureWritesDockerComposeFile(t *testing.T) {
	cfg, deps := setupDependencies(t)
	sensor, err := NewDockerSensor(context.Background(), deps, cfg, logging.NewTestLogger(t))
	if err != nil {
		t.Error(err)
	}

	assert.NotNil(t, sensor)
}

func TestImageStarts(t *testing.T) {
	cfg, deps := setupDependencies(t)
	sensor, err := NewDockerSensor(context.Background(), deps, cfg, logging.NewTestLogger(t))
	if err != nil {
		t.Error(err)
	}

	// Make sure we created the sensor
	assert.NotNil(t, sensor)
	cli := sensor.(*DockerConfig).manager.(*LocalDockerManager).dockerClient
	timeout := time.Now().Add(10 * time.Second)
	for {
		containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
		assert.NoError(t, err)
		if len(containers) > 0 {
			break
		}
		if time.Now().After(timeout) {
			t.Error("Timed out waiting for container to start")
		}
	}
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(containers))

	sensor.Close(context.Background())
}

func TestDownloadOnlyImages(t *testing.T) {
	cfg, deps := setupDependencies(t)
	cfg.ConvertedAttributes.(*Config).DownloadOnly = true
	sensor, err := NewDockerSensor(context.Background(), deps, cfg, logging.NewTestLogger(t))
	if err != nil {
		t.Error(err)
	}

	// Make sure we created the sensor
	assert.NotNil(t, sensor)

	// Now make sure it is only downloaded
	dm := sensor.(*DockerConfig).manager.(DockerManager)

	images, err := dm.ListImages()
	assert.NoError(t, err)

	// Search for image with RepoDigest "123"
	var foundImage *DockerImageDetails
	for _, image := range images {
		if image.RepoDigest == "sha256:218bb51abbd1864df8be26166f847547b3851a89999ca7bfceb85ca9b5d2e95d" {
			foundImage = &image
			break
		}
	}

	assert.True(t, foundImage != nil)

	// Assert that the image was found
	assert.NotNil(t, foundImage)

	containers, err := dm.ListContainers()
	assert.NoError(t, err)
	assert.Equal(t, 0, len(containers))

	sensor.Close(context.Background())
}

func TestRunOnce(t *testing.T) {
	cfg, deps := setupDependencies(t)
	cfg.ConvertedAttributes.(*Config).RunOnce = true

	sensor, err := NewDockerSensor(context.Background(), deps, cfg, logging.NewTestLogger(t))
	if err != nil {
		t.Error(err)
	}

	// Make sure we created the sensor
	assert.NotNil(t, sensor)

	// Now make sure it is actually running
	logger := logging.NewTestLogger(t)
	dm, err := NewLocalDockerManager(logger)
	assert.NoError(t, err)

	containers, err := dm.ListContainers()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(containers))

	docker := sensor.(*DockerConfig)
	assert.NotNil(t, docker)

	// Make sure the has run file has been updated.
	assert.Equal(t, 1, len(docker.containers))
	for _, container := range docker.containers {
		hasRun, err := container.GetHasRun()
		assert.NoError(t, err)
		assert.True(t, hasRun)
	}

	sensor.Close(context.Background())

	// Now create a new sensor with the same config and make sure it doesn't start a new container
	sensor, err = NewDockerSensor(context.Background(), deps, cfg, logging.NewTestLogger(t))
	if err != nil {
		t.Error(err)
	}

	// Make sure we created the sensor
	assert.NotNil(t, sensor)
	containers, err = dm.ListContainers()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(containers))
	sensor.Close(context.Background())
}
