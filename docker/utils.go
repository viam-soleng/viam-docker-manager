package docker

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"github.com/edaniels/golog"
)

func NewDockerImage(name string, tag string, repoDigest string, logger golog.Logger, cancelCtx context.Context, cancelFunc context.CancelFunc) *DockerImage {
	return &DockerImage{
		mu:         sync.RWMutex{},
		logger:     logger,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,
		Name:       name,
		RepoDigest: repoDigest,
		Tag:        tag,
	}
}

type DockerImage struct {
	mu         sync.RWMutex
	cancelCtx  context.Context
	cancelFunc context.CancelFunc
	logger     golog.Logger
	Name       string
	RepoDigest string
	Tag        string
}

func (di *DockerImage) Exists() bool {
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

func (di *DockerImage) Pull() error {
	di.logger.Debugf("Pulling image %s %s", di.Name, di.RepoDigest)
	proc := exec.Command("docker", "pull", fmt.Sprintf("%s:%s", di.Name, di.Tag))
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

func (di *DockerImage) IsRunning() (bool, error) {
	di.logger.Debugf("Checking if image %s %s is running", di.Name, di.RepoDigest)
	proc := exec.Command("docker", "ps", "-a")
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

	containerId, err := di.GetContainerId()
	if err != nil {
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

func (di *DockerImage) Start() error {
	di.mu.Lock()
	defer di.mu.Unlock()
	di.logger.Debugf("Starting image %s %s", di.Name, di.RepoDigest)

	args := make([]string, 0)
	args = append(args, "run", "--rm", "-d", "-it")
	args = append(args, fmt.Sprintf("%s@%s", di.Name, di.RepoDigest))
	// TODO: add support for passing in arguments
	args = append(args, "bash")
	proc := exec.Command("docker", args...)
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
	return nil
}

func (di *DockerImage) Stop() error {
	di.mu.Lock()
	defer di.mu.Unlock()
	di.logger.Debugf("Stopping image %s %s", di.Name, di.RepoDigest)

	containerId, err := di.GetContainerId()
	if err != nil {
		di.logger.Warn("Unable to get containerId.")
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
func (di *DockerImage) Remove() error {
	di.logger.Debugf("Removing image %s %s", di.Name, di.RepoDigest)
	imageId, err := di.GetImageId()
	if err != nil {
		di.logger.Warn("Unable to delete previous image.")
		return err
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

func (di *DockerImage) GetContainerId() (string, error) {
	imageId, err := di.GetImageId()
	if err != nil {
		di.logger.Warn("Unable to get ImageId.")
		return "", err
	}
	proc := exec.Command("docker", "container", "ls", "--all", fmt.Sprintf("--filter=ancestor=%s", imageId), "--format", "{{.ID}}")
	outputBytes, err := proc.Output()
	if err != nil {
		di.logger.Warn("Unable to get ContainerId.")
		exitError := err.(*exec.ExitError)
		if exitError != nil && exitError.Stderr != nil {
			di.logger.Errorf("Output: %s", string(exitError.Stderr))
		}
		di.logger.Error(err)
		return "", err
	}

	containerId := strings.TrimSpace(string(outputBytes))
	di.logger.Debugf("ContainerId: %s", containerId)
	return containerId, nil
}

func (di *DockerImage) GetImageId() (string, error) {
	proc := exec.Command("docker", "image", "inspect", "--format", "'{{json .Id}}'", fmt.Sprintf("%s@%s", di.Name, di.RepoDigest))
	outputBytes, err := proc.Output()
	if err != nil {
		exitError := err.(*exec.ExitError)
		if exitError != nil && exitError.Stderr != nil {
			di.logger.Errorf("Output: %s", string(exitError.Stderr))
		}
		di.logger.Error(err)
		return "", err
	}
	output := string(outputBytes)
	id := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(output, "\"", ""), "'", ""))
	di.logger.Debugf("ImageId: %s", id)
	return id, nil
}

func (image *DockerImage) Close() error {
	image.logger.Debugf("Closing image %s %s", image.Name, image.RepoDigest)
	return image.Stop()
}
