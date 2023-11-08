package docker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestListContainers(t *testing.T) {
	dm := LocalDockerManager{}
	containers, err := dm.ListContainers()
	if err != nil {
		t.Fatal(err)
	}

	for _, container := range containers {
		t.Logf("%#v", container)
	}
}

func TestListImages(t *testing.T) {
	dm := LocalDockerManager{}
	images, err := dm.ListImages()
	if err != nil {
		t.Fatal(err)
	}

	for _, image := range images {
		t.Logf("%#v", image)
	}
}

func TestGetContainerImageDigest(t *testing.T) {
	dm := LocalDockerManager{}
	digest, err := dm.GetContainerImageDigest("8ab34f2bc6e1d20825672e44be4252313503290abf160260070b776177e1d6be")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "sha256:2b7412e6465c3c7fc5bb21d3e6f1917c167358449fecac8176c6e496e5c1f05f", digest)
}

// TODO: This test will fail until we start the container. We need to add more setup code.
func TestGetContainersRunningImage(t *testing.T) {
	dm := LocalDockerManager{}
	containers, err := dm.GetContainersRunningImage("sha256:2b7412e6465c3c7fc5bb21d3e6f1917c167358449fecac8176c6e496e5c1f05f")
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 3, len(containers))
	for _, container := range containers {
		t.Logf("%#v", container)
	}
}
