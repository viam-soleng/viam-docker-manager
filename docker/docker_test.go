package docker

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
)

func setupDependencies() (resource.Config, resource.Dependencies) {
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

	return cfg, resource.Dependencies{}
}

func TestReconfigureWritesDockerComposeFile(t *testing.T) {
	cfg, deps := setupDependencies()
	sensor, err := NewDockerSensor(context.Background(), deps, cfg, logging.NewTestLogger(t))
	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, sensor)
}

func TestImageStarts(t *testing.T) {
	cfg, deps := setupDependencies()
	sensor, err := NewDockerSensor(context.Background(), deps, cfg, logging.NewTestLogger(t))
	if err != nil {
		t.Fatal(err)
	}

	// Make sure we created the sensor
	assert.NotNil(t, sensor)

	// Now make sure it is actually running
	dm := LocalDockerManager{}
	containers, err := dm.ListContainers()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(containers))

	sensor.Close(context.Background())
}

func TestCleanupOldImage(t *testing.T) {
	cfg, deps := setupDependencies()
	sensor, err := NewDockerSensor(context.Background(), deps, cfg, logging.NewTestLogger(t))
	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, sensor)

	dm := LocalDockerManager{}
	images, err := dm.ListImages()
	assert.NoError(t, err)
	if len(images) != 1 {
		t.Logf("Found %d images, expected 1", len(images))
		for _, image := range images {
			t.Logf("Image: %#v", image)
		}
		t.FailNow()
	}
	assert.Equal(t, 1, len(images))

	newConfig, _ := setupDependencies()
	newConfig.ConvertedAttributes.(*Config).RepoDigest = "sha256:c9cf959fd83770dfdefd8fb42cfef0761432af36a764c077aed54bbc5bb25368"
	newConfig.ConvertedAttributes.(*Config).ComposeFile = []string{
		"services:",
		"  app:",
		"    image: ubuntu@sha256:c9cf959fd83770dfdefd8fb42cfef0761432af36a764c077aed54bbc5bb25368",
		"    command: sleep 10",
		"    working_dir: /root",
	}
	sensor.Reconfigure(context.Background(), deps, newConfig)

	images, err = dm.ListImages()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(images))
	assert.Equal(t, "sha256:c9cf959fd83770dfdefd8fb42cfef0761432af36a764c077aed54bbc5bb25368", images[0].RepoDigest)

	err = dm.RemoveImageByRepoDigest("sha256:c9cf959fd83770dfdefd8fb42cfef0761432af36a764c077aed54bbc5bb25368")
	assert.NoError(t, err)
}

func TestDownloadOnlyImages(t *testing.T) {
	cfg, deps := setupDependencies()
	cfg.ConvertedAttributes.(*Config).DownloadOnly = true
	sensor, err := NewDockerSensor(context.Background(), deps, cfg, logging.NewTestLogger(t))
	if err != nil {
		t.Fatal(err)
	}

	// Make sure we created the sensor
	assert.NotNil(t, sensor)

	// Now make sure it is only downloaded
	dm := LocalDockerManager{}

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
	cfg, deps := setupDependencies()
	cfg.ConvertedAttributes.(*Config).RunOnce = true

	sensor, err := NewDockerSensor(context.Background(), deps, cfg, logging.NewTestLogger(t))
	if err != nil {
		t.Fatal(err)
	}

	// Make sure we created the sensor
	assert.NotNil(t, sensor)

	// Now make sure it is actually running
	dm := LocalDockerManager{}
	containers, err := dm.ListContainers()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(containers))

	docker := sensor.(*DockerConfig)
	assert.NotNil(t, docker)

	// Make sure the has run file has been updated.
	hasRun, err := docker.image.GetHasRun()
	assert.NoError(t, err)
	assert.True(t, hasRun)

	sensor.Close(context.Background())

	// Now create a new sensor with the same config and make sure it doesn't start a new container
	sensor, err = NewDockerSensor(context.Background(), deps, cfg, logging.NewTestLogger(t))
	if err != nil {
		t.Fatal(err)
	}

	// Make sure we created the sensor
	assert.NotNil(t, sensor)
	containers, err = dm.ListContainers()
	assert.NoError(t, err)
	assert.Equal(t, 1, len(containers))
	sensor.Close(context.Background())
}
