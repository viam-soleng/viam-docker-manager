package docker_deploy

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/docker/docker/client"
	"go.viam.com/rdk/logging"
)

type DockerContainer interface {
	IsRunning() (bool, error)
	GetHasRun() (bool, error)
	SetHasRun() error
	GetContainerId() string
	GetImageId() (string, error)
	GetRepoDigest() string
}

type LocalDockerContainer struct {
	mu           sync.RWMutex
	cancelCtx    context.Context
	logger       logging.Logger
	dockerClient *client.Client

	Id         string
	Name       string
	RepoDigest string
}

func NewDockerContainer(dockerClient *client.Client, containerId string, name string, repoDigest string, logger logging.Logger, cancelCtx context.Context) DockerContainer {
	return &LocalDockerContainer{
		mu:           sync.RWMutex{},
		logger:       logger,
		cancelCtx:    cancelCtx,
		dockerClient: dockerClient,
		Id:           containerId,
		Name:         name,
		RepoDigest:   repoDigest,
	}
}

func (di *LocalDockerContainer) IsRunning() (bool, error) {
	di.mu.Lock()
	defer di.mu.Unlock()
	di.logger.Debugf("Checking if container %s Image %s %s is running", di.Id, di.Name, di.RepoDigest)
	container, err := di.dockerClient.ContainerInspect(context.Background(), di.Id)
	if err != nil {
		return false, err
	}

	di.logger.Debugf("containerId: %v isRunning: %v", container.ID, container.State.Running)
	return container.State.Running, nil
}

func (di *LocalDockerContainer) GetContainerId() string {
	return di.Id
}

func (di *LocalDockerContainer) GetImageId() (string, error) {
	container, err := di.dockerClient.ContainerInspect(context.Background(), di.Id)
	if err != nil {
		return "", err
	}

	di.logger.Debugf("containerId: %v imageId: %v", container.ID, container.Image)
	return container.Image, nil
}

func (di *LocalDockerContainer) GetRepoDigest() string {
	return di.RepoDigest
}

func (di *LocalDockerContainer) readHasRunFile(f *os.File) (map[string]time.Time, error) {
	hasRunDict := make(map[string]time.Time)
	reader := csv.NewReader(f)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("unable to read has-run.status file: %w", err)
		}

		dockerImageID := record[0]
		lastRun, err := time.Parse(time.RFC3339, record[1])
		if err != nil {
			di.logger.Debugf("Unable to parse last run time: %v, Err: %w", record[1], err)
			continue
		}
		hasRunDict[dockerImageID] = lastRun
	}
	di.logger.Debugf("hasRunDict: %v", hasRunDict)
	return hasRunDict, nil
}

func (di *LocalDockerContainer) GetHasRun() (bool, error) {
	hasRunFile, err := getHasRunStatusFileHandle()
	if err != nil {
		return false, err
	}

	defer closeHasRunStatus(hasRunFile)

	hasRunDict, err := di.readHasRunFile(hasRunFile)
	if err != nil {
		return false, err
	}
	// Check if hasRunDict contains the key dc.image.GetRepoDigest()
	if lastRun, ok := hasRunDict[di.GetRepoDigest()]; ok {
		// Key exists in hasRunDict
		di.logger.Debugf("Image has run before: %v", lastRun)
		// The image has run before
		return true, nil
	} else {
		// The image has not run before
		return false, nil
	}
}

func (di *LocalDockerContainer) SetHasRun() error {
	hasRunFile, err := getHasRunStatusFileHandle()
	if err != nil {
		return err
	}

	defer closeHasRunStatus(hasRunFile)

	hasRunDict, err := di.readHasRunFile(hasRunFile)
	if err != nil {
		return err
	}

	// The image has not run before
	hasRunDict[di.GetRepoDigest()] = time.Now()
	hasRunFile.Truncate(0)
	hasRunFile.Seek(0, 0)
	writer := csv.NewWriter(hasRunFile)
	for k, v := range hasRunDict {
		err := writer.Write([]string{k, v.Format(time.RFC3339)})
		if err != nil {
			return fmt.Errorf("unable to write has-run.status file: %w", err)
		}
	}
	writer.Flush()
	return nil
}

func getHasRunStatusFileHandle() (*os.File, error) {
	moduleDirectory := os.Getenv("VIAM_MODULE_DATA")
	if moduleDirectory == "" {
		return nil, errors.New("VIAM_MODULE_DATA is not set")
	}

	hasRunFilePath := fmt.Sprintf("%s/%s", moduleDirectory, "has-run.status")
	hasRunFile, err := os.OpenFile(hasRunFilePath, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("unable to open has-run.status file: %w", err)
	}

	// Lock the file to make sure nobody messes with it while we're in startup.
	err = syscall.Flock(int(hasRunFile.Fd()), syscall.LOCK_EX)
	if err != nil {
		// Make sure we close the file if we fail to lock it
		defer hasRunFile.Close()
		return nil, fmt.Errorf("unable to lock has-run.status file: %w", err)
	}
	return hasRunFile, nil
}

func closeHasRunStatus(hasRunFile *os.File) {
	// Don't forget to unlock the file when we're done.
	defer syscall.Flock(int(hasRunFile.Fd()), syscall.LOCK_UN)
	defer hasRunFile.Close()
}
