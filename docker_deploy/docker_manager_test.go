package docker_deploy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.viam.com/rdk/logging"
)

func TestListContainers(t *testing.T) {
	logger := logging.NewTestLogger(t)
	dm, err := NewLocalDockerManager(logger)
	assert.NoError(t, err)
	containers, err := dm.ListContainers()
	if err != nil {
		t.Fatal(err)
	}

	for _, container := range containers {
		t.Logf("%#v", container)
	}
}

func TestListImages(t *testing.T) {
	logger := logging.NewTestLogger(t)
	dm, err := NewLocalDockerManager(logger)
	assert.NoError(t, err)

	images, err := dm.ListImages()
	if err != nil {
		t.Fatal(err)
	}

	for _, image := range images {
		t.Logf("%#v", image)
	}
}

func TestGetContainerImageDigest(t *testing.T) {
	logger := logging.NewTestLogger(t)
	dm, err := NewLocalDockerManager(logger)
	assert.NoError(t, err)
	digest, err := dm.GetContainerImageDigest("8ab34f2bc6e1d20825672e44be4252313503290abf160260070b776177e1d6be")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "sha256:2b7412e6465c3c7fc5bb21d3e6f1917c167358449fecac8176c6e496e5c1f05f", digest)
}

// TODO: This test will fail until we start the container. We need to add more setup code.
func TestGetContainersRunningImage(t *testing.T) {
	logger := logging.NewTestLogger(t)
	dm, err := NewLocalDockerManager(logger)
	assert.NoError(t, err)
	containers, err := dm.GetContainersRunningImage("sha256:2b7412e6465c3c7fc5bb21d3e6f1917c167358449fecac8176c6e496e5c1f05f")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 3, len(containers))
	for _, container := range containers {
		t.Logf("%#v", container)
	}
}

func TestImageExists(t *testing.T) {
	logger := logging.NewTestLogger(t)
	dm, err := NewLocalDockerManager(logger)
	assert.NoError(t, err)

	err = dm.PullImage("mcr.microsoft.com/dotnet/samples", "sha256:test")
	assert.NoError(t, err)
	exists, err := dm.ImageExists("sha256:test")
	assert.NoError(t, err)
	assert.False(t, exists)

	err = dm.PullImage("mcr.microsoft.com/dotnet/samples", "sha256:test")
	assert.NoError(t, err)
	exists, err = dm.ImageExists("sha256:d41fe80991d7c26ad43b052bb87c68a216a365c143623a62b5a5963fcdb77eb1")
	assert.NoError(t, err)
	assert.True(t, exists)

	t.Cleanup(func() {
		err = dm.RemoveImageByImageId("sha256:d41fe80991d7c26ad43b052bb87c68a216a365c143623a62b5a5963fcdb77eb1")
		assert.NoError(t, err)
	})
}

func TestImagePull(t *testing.T) {
	logger := logging.NewTestLogger(t)
	dm, err := NewLocalDockerManager(logger)
	assert.NoError(t, err)

	exists, err := dm.ImageExists("sha256:d41fe80991d7c26ad43b052bb87c68a216a365c143623a62b5a5963fcdb77eb1")
	assert.NoError(t, err)
	assert.False(t, exists)

	err = dm.PullImage("mcr.microsoft.com/dotnet/samples", "sha256:test")
	assert.NoError(t, err)
	exists, err = dm.ImageExists("sha256:d41fe80991d7c26ad43b052bb87c68a216a365c143623a62b5a5963fcdb77eb1")
	assert.NoError(t, err)
	assert.True(t, exists)

	t.Cleanup(func() {
		err = dm.RemoveImageByImageId("sha256:d41fe80991d7c26ad43b052bb87c68a216a365c143623a62b5a5963fcdb77eb1")
		assert.NoError(t, err)
	})
}

func TestImageRemove(t *testing.T) {
	logger := logging.NewTestLogger(t)
	dm, err := NewLocalDockerManager(logger)
	assert.NoError(t, err)

	exists, err := dm.ImageExists("sha256:d41fe80991d7c26ad43b052bb87c68a216a365c143623a62b5a5963fcdb77eb1")
	assert.NoError(t, err)
	assert.False(t, exists)

	err = dm.PullImage("mcr.microsoft.com/dotnet/samples", "sha256:test")
	assert.NoError(t, err)
	exists, err = dm.ImageExists("sha256:d41fe80991d7c26ad43b052bb87c68a216a365c143623a62b5a5963fcdb77eb1")
	assert.NoError(t, err)
	assert.True(t, exists)

	err = dm.RemoveImageByImageId("sha256:d41fe80991d7c26ad43b052bb87c68a216a365c143623a62b5a5963fcdb77eb1")
	assert.NoError(t, err)

	exists, err = dm.ImageExists("sha256:d41fe80991d7c26ad43b052bb87c68a216a365c143623a62b5a5963fcdb77eb1")
	assert.NoError(t, err)
	assert.False(t, exists)
}
