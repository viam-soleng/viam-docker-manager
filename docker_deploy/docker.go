package docker_deploy

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.viam.com/rdk/components/sensor"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	viamutils "go.viam.com/utils"
)

var Model = resource.NewModel("viam-soleng", "manage", "docker")

type DockerConfig struct {
	resource.Named
	mu           sync.RWMutex
	logger       logging.Logger
	cancelCtx    context.Context
	cancelFunc   func()
	containers   []DockerContainer
	manager      DockerManager
	watchers     []func()
	stop         chan bool
	wg           sync.WaitGroup
	downloadOnly bool
	runOnce      bool
	conf         Config
}

func init() {
	resource.RegisterComponent(
		sensor.API,
		Model,
		resource.Registration[sensor.Sensor, *Config]{Constructor: NewDockerSensor})
}

func NewDockerSensor(ctx context.Context, deps resource.Dependencies, conf resource.Config, logger logging.Logger) (sensor.Sensor, error) {
	logger.Info("Starting Docker Manager Module v0.0.2")
	cancelCtx, cancelFunc := context.WithCancel(context.Background())
	b := DockerConfig{
		Named:      conf.ResourceName().AsNamed(),
		logger:     logger,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,
		mu:         sync.RWMutex{},
		stop:       make(chan bool, 1),
		wg:         sync.WaitGroup{},
		containers: []DockerContainer{},
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
	defer func() {
		dc.conf = *newConf
	}()

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

	// Let's try to be efficient and only make changes if changes happened.
	if !dc.conf.HasChanged(newConf) {
		return nil
	}

	if dc.manager == nil {
		if newConf.Credentials != nil {
			manager, err := NewLocalDockerManagerWithAuth(newConf.Credentials.Username, newConf.Credentials.Password, dc.logger)
			if err != nil {
				return err
			}
			dc.manager = manager
		} else {
			manager, err := NewLocalDockerManager(dc.logger)
			if err != nil {
				return err
			}
			dc.manager = manager
		}
	}

	// Close the existing containers, remove it, and set it to nil
	// Should download the new image before stopping the old one
	if len(dc.containers) > 0 {
		for _, container := range dc.containers {
			dc.manager.StopContainer(container.GetContainerId())
			dc.manager.RemoveContainer(container.GetContainerId())
		}
		dc.containers = []DockerContainer{}
	}

	// TODO: Cleanup old images
	// Possibly tag images with the component name that uses them?

	// Make sure we track if the image is download only
	// I'm not a huge fan of this functionality, it feels like we're using the wrong tool for
	// the job, but it's what we have for now.
	dc.downloadOnly = newConf.DownloadOnly

	// Make sure we track if the image is run once only
	dc.runOnce = newConf.RunOnce

	// Check if the image exists locally already
	repoDigest, err := newConf.GetRepoDigest()
	if err != nil {
		return err
	}
	imageExists, err := dc.manager.ImageExists(repoDigest)
	if err != nil {
		return err
	}

	imageName, err := newConf.GetImageName()
	if err != nil {
		return err
	}
	// If the image doesn't exist, pull it
	if !imageExists {
		dc.logger.Infof("Image %s does not exist. Pulling...", imageName)
		err := dc.manager.PullImage(imageName, repoDigest)
		if err != nil {
			return err
		}
	}

	if !dc.downloadOnly {
		if newConf.ComposeOptions != nil {
			containers, err := dc.manager.CreateComposeContainers(newConf.ComposeOptions.ImageName, newConf.ComposeOptions.RepoDigest, newConf.ComposeOptions.ComposeFile, dc.logger, dc.cancelCtx, dc.cancelFunc)
			if err != nil {
				return err
			}
			dc.containers = containers
		} else if newConf.RunOptions != nil {
			container, err := dc.manager.CreateContainer(newConf.RunOptions.ImageName, newConf.RunOptions.RepoDigest, newConf.RunOptions.EntryPointArgs, newConf.RunOptions.Options, dc.logger, dc.cancelCtx, dc.cancelFunc)
			if err != nil {
				return err
			}
			dc.containers = []DockerContainer{container}
		} else {
			// In theory this is impossible to hit as long as Validate is called and the returned errors are handled properly, but we'll leave it here just in case
			return errors.New("no run options or compose options specified")
		}
	}

	if dc.watchers == nil {
		dc.watchers = make([]func(), len(dc.containers))
		for i, container := range dc.containers {
			dc.watchers[i] = func() {
				dc.wg.Add(1)
				defer dc.wg.Done()
				for {
					select {
					case <-dc.cancelCtx.Done():
						dc.logger.Info("received cancel signal")
						return
					case <-dc.stop:
						dc.logger.Info("received stop signal")
						return
					case <-time.After(10 * time.Second):
						dc.logger.Debug("iterating...")
					}

					dc.logger.Debug("image exists. Checking if running...")
					isRunning, err := container.IsRunning()
					if err != nil {
						dc.logger.Error(err)
						continue
					}
					if !isRunning && dc.shouldRun() {
						dc.logger.Debug("container not running. Starting...")
						dc.startInternal()
					} else {
						dc.logger.Debug("container run conditions satisfied. Sleeping...")
					}
				}
			}
			viamutils.PanicCapturingGo(dc.watchers[i])
		}

	}
	return nil
}

func (dc *DockerConfig) getReadings(container DockerContainer) (map[string]interface{}, error) {
	imageId, err := container.GetImageId()
	if err != nil {
		return nil, err
	}

	if imageId == "" {
		return nil, errors.New("imageId is empty")
	}
	isRunning, err := container.IsRunning()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"repoDigest":  container.GetRepoDigest(),
		"imageId":     imageId,
		"containerId": container.GetContainerId(),
		"isRunning":   isRunning,
	}, nil
}

// Readings implements sensor.Sensor.
func (dc *DockerConfig) Readings(ctx context.Context, extra map[string]interface{}) (map[string]interface{}, error) {
	resp := map[string]interface{}{}
	for _, container := range dc.containers {
		readings, err := dc.getReadings(container)
		if err != nil {
			dc.logger.Error(err)
			continue
		}
		resp[container.GetContainerId()] = readings
	}
	return resp, nil
}

func (dc *DockerConfig) Close(ctx context.Context) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.logger.Debug("Closing Docker Manager Module")
	dc.stop <- true
	for _, container := range dc.containers {
		if container != nil {
			dc.logger.Debugf("Stopping container %v", container.GetContainerId())
			err := dc.manager.StopContainer(container.GetContainerId())
			if err != nil {
				dc.logger.Error(err)
			}
		}
	}
	dc.logger.Debug("Stop command sent, waiting on WaitGroup")
	dc.wg.Wait()
	return nil
}

func (dc *DockerConfig) Ready(ctx context.Context, extra map[string]interface{}) (bool, error) {
	isRunning := false
	for _, container := range dc.containers {
		h, err := container.IsRunning()
		if err != nil {
			dc.logger.Error(err)
		}
		isRunning = isRunning && h
	}
	return isRunning, nil
}

func (dc *DockerConfig) shouldRun() bool {
	// If the image is only configured to be downloaded, we don't want to start it
	if dc.downloadOnly {
		return false
	}
	// If the image should run once only, we don't want to start it if it has already run
	hasRun := false
	for _, container := range dc.containers {
		h, err := container.GetHasRun()
		if err != nil {
			dc.logger.Error(err)
		}
		hasRun = hasRun && h
	}
	if (dc.runOnce && !hasRun) || !dc.runOnce {
		return true
	}
	return false
}

func (dc *DockerConfig) startInternal() {
	for _, container := range dc.containers {
		dc.logger.Debug("Starting container %v", container.GetContainerId())
		err := dc.manager.StartContainer(container.GetContainerId())
		if err != nil {
			dc.logger.Error(err)
		}
		container.SetHasRun()
	}
}
