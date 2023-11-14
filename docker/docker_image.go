package docker

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"go.viam.com/rdk/logging"
)

type DockerImage interface {
	Exists() bool
	Start() error
	Stop() error
	Remove() error
	IsRunning() (bool, error)
	GetImageId() string
	GetContainerId() string
	GetRepoDigest() string
}

func NewLocalDockerImage() DockerImage {

	return &LocalDockerImage{}
}

type LocalDockerImage struct {
	mu             sync.RWMutex
	cancelCtx      context.Context
	cancelFunc     context.CancelFunc
	logger         logging.Logger
	Name           string
	RepoDigest     string
	ComposeFile    string
	EntryPointArgs []string
	Options        []string
}

func NewDockerComposeImage(name string, repoDigest string, composeFile string, logger logging.Logger, cancelCtx context.Context, cancelFunc context.CancelFunc) DockerImage {
	return &LocalDockerImage{
		mu:          sync.RWMutex{},
		logger:      logger,
		cancelCtx:   cancelCtx,
		cancelFunc:  cancelFunc,
		Name:        name,
		RepoDigest:  repoDigest,
		ComposeFile: composeFile,
	}
}

func NewDockerImage(name string, repoDigest string, entry_point_args []string, options []string, logger logging.Logger, cancelCtx context.Context, cancelFunc context.CancelFunc) DockerImage {
	return &LocalDockerImage{
		mu:             sync.RWMutex{},
		logger:         logger,
		cancelCtx:      cancelCtx,
		cancelFunc:     cancelFunc,
		Name:           name,
		RepoDigest:     repoDigest,
		ComposeFile:    "",
		EntryPointArgs: entry_point_args,
		Options:        options,
	}
}

func (di *LocalDockerImage) Exists() bool {
	di.logger.Debugf("Checking if image %s %s exists", di.Name, di.RepoDigest)
	proc := exec.Command("docker", "images", "--digests")
	outputBytes, err := proc.Output()
	if err != nil {
		exitError := err.(*exec.ExitError)
		if exitError != nil && exitError.Stderr != nil {
			di.logger.Errorf("Output: %s", string(exitError.Stderr))
		}
		di.logger.Error(err)
		return false
	}
	output := string(outputBytes)
	di.logger.Debugf("Output: %s", output)
	if strings.Contains(output, "Error: No such image") {
		return false
	}
	return strings.Contains(output, di.RepoDigest)
}

func (di *LocalDockerImage) IsRunning() (bool, error) {
	di.logger.Debugf("Checking if image %s %s is running", di.Name, di.RepoDigest)
	proc := exec.Command("docker", "ps", "--no-trunc")
	outputBytes, err := proc.Output()
	if err != nil {
		exitError := err.(*exec.ExitError)
		if exitError != nil && exitError.Stderr != nil {
			di.logger.Errorf("Output: %s", string(exitError.Stderr))
		}
		di.logger.Error(err)
	}
	outputString := string(outputBytes)
	di.logger.Debugf("Output: %s", outputString)

	containerId := di.GetContainerId()
	if containerId == "" {
		di.logger.Warn("Unable to get containerId.")
		return false, err
	}
	lines := strings.Split(outputString, "\n")
	for _, line := range lines {
		if strings.Contains(line, containerId) && strings.Contains(line, "Up") {
			return true, nil
		}
	}

	return false, nil
}

func (di *LocalDockerImage) Start() error {
	di.mu.Lock()
	defer di.mu.Unlock()
	di.logger.Debugf("Starting image %s %s", di.Name, di.RepoDigest)
	args := make([]string, 0)
	if di.ComposeFile == "" {
		args = append(args, "run", "--rm", "-d")
		args = append(args, di.Options...)
		args = append(args, fmt.Sprintf("%s@%s", di.Name, di.RepoDigest))
		args = append(args, di.EntryPointArgs...)
	} else {
		args = append(args, "compose", "-f", di.ComposeFile, "up", "-d")
	}
	proc := exec.Command("docker", args...)
	di.logger.Warn(proc.String())
	outputBytes, err := proc.Output()
	if err != nil {
		exitError := err.(*exec.ExitError)
		if exitError != nil && exitError.Stderr != nil {
			di.logger.Errorf("Output: %s", string(exitError.Stderr))
		}
		di.logger.Error(err)
	}
	outputString := string(outputBytes)

	di.logger.Debugf("Output: %s", outputString)
	di.logger.Debug("Done starting container")
	return nil
}

func (di *LocalDockerImage) Stop() error {
	di.mu.Lock()
	defer di.mu.Unlock()
	di.logger.Debugf("Stopping image %s %s", di.Name, di.RepoDigest)

	containerId := di.GetContainerId()
	if containerId == "" {
		di.logger.Warn("Unable to get containerId.")
		return errors.New("unable to stop image")
	}

	proc := exec.Command("docker", "stop", containerId)
	outputBytes, err := proc.Output()
	if err != nil {
		di.logger.Warn("Unable to stop image.")
		exitError := err.(*exec.ExitError)
		if exitError != nil && exitError.Stderr != nil {
			di.logger.Errorf("Output: %s", string(exitError.Stderr))
		}
		di.logger.Error(err)
		return err
	}
	outputString := string(outputBytes)
	di.logger.Debugf("Output: %s", outputString)
	return nil
}

// TODO: I think this doesn't make sense, I think maybe I need to make this a static method to find and delete unused images
func (di *LocalDockerImage) Remove() error {
	di.logger.Debugf("Removing image %s %s", di.Name, di.RepoDigest)
	imageId := di.GetImageId()
	if imageId == "" {
		di.logger.Warn("Unable to delete previous image.")
		return errors.New("unable to delete previous image")
	}
	proc := exec.Command("docker", "rmi", imageId)
	// TODO: Read output from proc using a pipe
	// output:=proc.StdoutPipe()

	outputBytes, err := proc.Output()
	if err != nil {
		exitError := err.(*exec.ExitError)
		if exitError != nil && exitError.Stderr != nil {
			di.logger.Errorf("Output: %s", string(exitError.Stderr))
		}
		di.logger.Error(err)
		return err
	}
	di.logger.Debugf("Output: %s", string(outputBytes))
	return nil
}

func (di *LocalDockerImage) GetContainerId() string {
	imageId := di.GetImageId()
	if imageId == "" {
		di.logger.Warn("Unable to get ImageId.")
		return ""
	}
	proc := exec.Command("docker", "container", "ls", "--all", fmt.Sprintf("--filter=ancestor=%s", imageId), "--format", "{{.ID}}", "--no-trunc")
	outputBytes, err := proc.Output()
	if err != nil {
		di.logger.Warn("Unable to get ContainerId.")
		di.logger.Error(err)
		exitError := err.(*exec.ExitError)
		if exitError != nil && exitError.Stderr != nil {
			di.logger.Errorf("Output: %s", string(exitError.Stderr))
		}
		return ""
	}

	containerId := strings.TrimSpace(string(outputBytes))
	di.logger.Debugf("ContainerId: %s", containerId)
	return containerId
}

func (di *LocalDockerImage) GetImageId() string {
	proc := exec.Command("docker", "image", "inspect", "--format", "'{{json .Id}}'", fmt.Sprintf("%s@%s", di.Name, di.RepoDigest))
	outputBytes, err := proc.Output()
	if err != nil {
		exitError := err.(*exec.ExitError)
		if exitError != nil && exitError.Stderr != nil {
			// This is a special case, if the image does not exist, that should be fine-ish
			if strings.Contains(string(exitError.Stderr), "Error: No such image") {
				di.logger.Error(ErrImageDoesNotExist)
				return ""
			}
			di.logger.Errorf("Output: %s", string(exitError.Stderr))
		}
		di.logger.Error(err)
		return ""
	}
	output := string(outputBytes)
	id := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(output, "\"", ""), "'", ""))
	di.logger.Debugf("ImageId: %s", id)
	return id
}

func (di *LocalDockerImage) GetRepoDigest() string {
	return di.RepoDigest
}
