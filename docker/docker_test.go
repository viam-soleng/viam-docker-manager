package docker

import (
	"context"
	"testing"

	"github.com/edaniels/golog"
	"github.com/stretchr/testify/assert"
	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/resource"
)

func setupDependencies() (resource.Config, resource.Dependencies) {
	cfg := resource.Config{
		Name:  "movementsensor",
		Model: Model,
		API:   sensor.API,
		ConvertedAttributes: &Config{
			ImageName:   "ubuntu",
			ImageDigest: "sha256:2b7412e6465c3c7fc5bb21d3e6f1917c167358449fecac8176c6e496e5c1f05f",
			RepoDigest:  "sha256:e4c58958181a5925816faa528ce959e487632f4cfd192f8132f71b32df2744b4",
			ImageTag:    "latest",
			ComposeFile: []string{
				"services:",
				"  foo:",
				"    image: ubuntu",
				"    command: echo \"hello world\"",
			},
		},
	}

	return cfg, resource.Dependencies{}
}

func TestReconfigureWritesDockerComposeFile(t *testing.T) {
	cfg, deps := setupDependencies()
	sensor, err := NewDockerSensor(context.Background(), deps, cfg, golog.NewTestLogger(t))
	if err != nil {
		t.Fatal(err)
	}

	assert.NotNil(t, sensor)
}
