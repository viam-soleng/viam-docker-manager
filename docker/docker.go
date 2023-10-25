package docker

import (
	"context"
	"errors"
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
	imageUrl   string
	dockerPath string
}

func init() {
	resource.RegisterComponent(
		sensor.API,
		Model,
		resource.Registration[sensor.Sensor, *Config]{Constructor: NewDockerSensor})
}

func NewDockerSensor(ctx context.Context, deps resource.Dependencies, conf resource.Config, logger golog.Logger) (sensor.Sensor, error) {
	logger.Info("Starting Applied Motion Products ST Motor Driver v0.1")
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

func (b *DockerConfig) Reconfigure(ctx context.Context, _ resource.Dependencies, conf resource.Config) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.logger.Debug("Reconfiguring Docker Manager Module")

	newConf, err := resource.NativeConfig[*Config](conf)
	if err != nil {
		return err
	}

	// In case the module has changed name
	b.Named = conf.ResourceName().AsNamed()

	// Check if the image exists already?
	// If image exists and is running, return
	// If image exists and is not running, start it.
	// If image does not exist, pull it
	// Start image
	image := NewDockerImage(newConf.ImageName, newConf.ImageTag, newConf.RepoDigest, b.logger, b.cancelCtx, b.cancelFunc)
	if !image.Exists() {
		err := image.Pull()
		if err != nil {
			return err
		}
		return image.Start()
	} else {
		isRunning, err := image.IsRunning()
		if err != nil {
			return err
		}
		if !isRunning {
			return image.Start()
		}
	}

	return nil
}

// Readings implements sensor.Sensor.
func (*DockerConfig) Readings(ctx context.Context, extra map[string]interface{}) (map[string]interface{}, error) {
	foo := map[string]interface{}{
		"isRunning":      true,
		"sha256":         "",
		"container_path": "",
	}
	return foo, nil
}

func (s *DockerConfig) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return errors.New("not implemented")
	// TODO: Shut down the container?
}
