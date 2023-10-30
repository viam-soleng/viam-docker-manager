package docker

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/edaniels/golog"
	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/resource"
)

var Model = resource.NewModel("viam-soleng", "manage", "docker")

type DockerConfig struct {
	resource.Named
	mu         sync.RWMutex
	logger     golog.Logger
	cancelCtx  context.Context
	cancelFunc func()
	image      *DockerImage
}

func init() {
	resource.RegisterComponent(
		sensor.API,
		Model,
		resource.Registration[sensor.Sensor, *Config]{Constructor: NewDockerSensor})
}

func NewDockerSensor(ctx context.Context, deps resource.Dependencies, conf resource.Config, logger golog.Logger) (sensor.Sensor, error) {
	logger.Info("Starting Docker Manager Module v0.0.1")
	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	b := DockerConfig{
		Named:      conf.ResourceName().AsNamed(),
		logger:     logger,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,
		mu:         sync.RWMutex{},
	}

	if err := b.Reconfigure(ctx, deps, conf); err != nil {
		return nil, err
	}
	return &b, nil
}

func (dc *DockerConfig) Reconfigure(ctx context.Context, _ resource.Dependencies, conf resource.Config) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.logger.Debug("Reconfiguring Docker Manager Module")

	newConf, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return err
	}

	// In case the module has changed name
	dc.Named = conf.ResourceName().AsNamed()

	return dc.reconfigure(newConf)
}

// A helper function to reconfigure the module, broken out from Reconfigure to make testing easier.
func (dc *DockerConfig) reconfigure(newConf *Config) error {
	// Check if the image exists already?
	// If image exists and is running, return
	// If image exists and is not running, start it.
	// If image does not exist, pull it
	// Start image

	// Close the existing image
	if dc.image != nil {
		dc.image.Close()
	}

	// delete the old compose file if the new config doesn't have one

	// where to store the compose file? maybe in the DockerImage?
	// Write the new compose file
	composeFile := ""
	if newConf.ComposeFile != nil {
		composeFile := strings.Replace(newConf.ImageName, "/", "-", -1)
		path := fmt.Sprintf("%s/%s-%s.yml", os.TempDir(), "docker-compose", composeFile)
		dc.logger.Info("Writing docker-compose file %s", path)
		fs, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}
		defer fs.Close()
		for _, line := range newConf.ComposeFile {
			fs.WriteString(fmt.Sprintln(line))
		}
	}

	dc.image = NewDockerImage(newConf.ImageName, newConf.ImageTag, newConf.RepoDigest, composeFile, dc.logger, dc.cancelCtx, dc.cancelFunc)
	if !dc.image.Exists() {
		err := dc.image.Pull()
		if err != nil {
			return err
		}
		return dc.image.Start()
	} else {
		isRunning, err := dc.image.IsRunning()
		if err != nil {
			return err
		}
		if !isRunning {
			return dc.image.Start()
		} else {
			return nil
		}
	}
}

// Readings implements sensor.Sensor.
func (dc *DockerConfig) Readings(ctx context.Context, extra map[string]interface{}) (map[string]interface{}, error) {
	if dc.image == nil {
		return map[string]interface{}{
			"isRunning":  false,
			"repoDigest": "",
			"imageTag":   "",
			"imageId":    "",
		}, nil
	}

	imageId, err := dc.image.GetImageId()
	if err != nil {
		return nil, err
	}
	isRunning, err := dc.image.IsRunning()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"repoDigest": dc.image.RepoDigest,
		"imageTag":   dc.image.Tag,
		"imageId":    imageId,
		"isRunning":  isRunning,
	}, nil
}

func (dc *DockerConfig) Close(ctx context.Context) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.logger.Debug("Closing Docker Manager Module")
	if dc.image != nil {
		return dc.image.Close()
	}
	return nil
}
