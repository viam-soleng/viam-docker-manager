package docker

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
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
	image        DockerImage
	manager      DockerManager
	watcher      func()
	stop         chan bool
	wg           sync.WaitGroup
	downloadOnly bool
	runOnce      bool
	hasRun       bool
}

func init() {
	resource.RegisterComponent(
		sensor.API,
		Model,
		resource.Registration[sensor.Sensor, *Config]{Constructor: NewDockerSensor})
}

func NewDockerSensor(ctx context.Context, deps resource.Dependencies, conf resource.Config, logger logging.Logger) (sensor.Sensor, error) {
	logger.Info("Starting Docker Manager Module v0.0.1")
	cancelCtx, cancelFunc := context.WithCancel(context.Background())

	b := DockerConfig{
		Named:      conf.ResourceName().AsNamed(),
		logger:     logger,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,
		mu:         sync.RWMutex{},
		manager:    NewLocalDockerManager(logger),
		stop:       make(chan bool),
		wg:         sync.WaitGroup{},
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
		dc.image.Stop()
	}

	// delete the old compose file if the new config doesn't have one
	dm := LocalDockerManager{}
	if dc.image != nil && newConf.RepoDigest != "" && dc.image.GetRepoDigest() != newConf.RepoDigest {
		err := dm.RemoveImageByRepoDigest(dc.image.GetRepoDigest())
		if err != nil {
			dc.logger.Warnf("Error removing old image: %v", err)
		}
	}

	// where to store the compose file? maybe in the DockerImage?
	// Write the new compose file
	if newConf.ComposeFile != nil {
		sanitizedImageName := strings.Replace(newConf.ImageName, "/", "-", -1)
		composeFile := fmt.Sprintf("%s/%s-%s.yml", os.TempDir(), "docker-compose", sanitizedImageName)
		dc.logger.Infof("Writing docker-compose file %s", composeFile)
		fs, err := os.OpenFile(composeFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
		if err != nil {
			return err
		}
		defer fs.Close()
		for _, line := range newConf.ComposeFile {
			_, err := fs.WriteString(fmt.Sprintln(line))
			if err != nil {
				return err
			}
		}
		dc.image = NewDockerComposeImage(newConf.ImageName, newConf.RepoDigest, composeFile, dc.logger, dc.cancelCtx, dc.cancelFunc)
	} else {
		dc.image = NewDockerImage(newConf.ImageName, newConf.RepoDigest, newConf.EntryPointArgs, newConf.Options, dc.logger, dc.cancelCtx, dc.cancelFunc)
	}

	// Make sure we track if the image is download only
	// I'm not a huge fan of this functionality, it feels like we're using the wrong tool for
	// the job, but it's what we have for now.
	dc.downloadOnly = newConf.DownloadOnly

	// Make sure we track if the image is run once only
	dc.runOnce = newConf.RunOnce

	// For any image that is configured to run once only, we need to track if it has run already
	hasRun, err := dc.image.GetHasRun()
	if err != nil {
		dc.logger.Warnf("Error getting hasRun: %v", err)
	}
	dc.hasRun = hasRun

	if dc.watcher == nil {
		dc.watcher = func() {
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
				default:
					if !dc.image.Exists() {
						dc.logger.Debug("image does not exist. Pulling...")
						err := dc.manager.PullImage(newConf.ImageName, newConf.RepoDigest)
						if err != nil {
							dc.logger.Error(err)
							continue
						}
						if dc.shouldRun() {
							dc.logger.Debug("image pulled. Starting...")
							dc.startInternal()
						}
					} else {
						dc.logger.Debug("image exists. Checking if running...")
						isRunning, err := dc.image.IsRunning()
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

				time.Sleep(10 * time.Second)
			}
		}
		viamutils.PanicCapturingGo(dc.watcher)
	}
	return nil
}

// Readings implements sensor.Sensor.
func (dc *DockerConfig) Readings(ctx context.Context, extra map[string]interface{}) (map[string]interface{}, error) {
	if dc.image == nil {
		return map[string]interface{}{
			"repoDigest":  "",
			"imageId":     "",
			"containerId": "",
			"isRunning":   false,
		}, nil
	}

	imageId := dc.image.GetImageId()
	if imageId == "" {
		return nil, errors.New("imageId is empty")
	}
	isRunning, err := dc.image.IsRunning()
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"repoDigest":  dc.image.GetRepoDigest(),
		"imageId":     imageId,
		"containerId": dc.image.GetContainerId(),
		"isRunning":   isRunning,
	}, nil
}

func (dc *DockerConfig) Close(ctx context.Context) error {
	dc.mu.Lock()
	defer dc.mu.Unlock()
	dc.logger.Debug("Closing Docker Manager Module")
	dc.stop <- true
	if dc.image != nil {
		return dc.image.Stop()
	}
	dc.wg.Wait()
	return nil
}

func (dc *DockerConfig) Ready(ctx context.Context, extra map[string]interface{}) (bool, error) {
	return dc.image.IsRunning()
}

func (dc *DockerConfig) shouldRun() bool {
	// If the image is only configured to be downloaded, we don't want to start it
	if dc.downloadOnly {
		return false
	}
	// If the image should run once only, we don't want to start it if it has already run
	if (dc.runOnce && !dc.hasRun) || !dc.runOnce {
		return true
	}
	return false
}

func (dc *DockerConfig) startInternal() {
	err := dc.image.Start()
	if err != nil {
		dc.logger.Error(err)
	}
	dc.image.SetHasRun()
}
