package docker

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/edaniels/golog"
)

func (image *DockerImage) Exists() bool {
	image.logger.Debugf("Checking if image %s %s exists", image.Name, image.RepoDigest)
	proc := exec.Command("docker", "images", "--digests")
	outputBytes, err := proc.Output()
	if err != nil {
		exitError := err.(*exec.ExitError)
		if exitError != nil && exitError.Stderr != nil {
			image.logger.Errorf("Output: %s", string(exitError.Stderr))
		}
		image.logger.Error(err)
		return false
	}
	output := string(outputBytes)
	image.logger.Debugf("Output: %s", output)
	if strings.Contains(output, "Error: No such image") {
		return false
	}
	return strings.Contains(output, image.RepoDigest)
}

func (image *DockerImage) Pull() error {
	image.logger.Debugf("Pulling image %s %s", image.Name, image.RepoDigest)
	proc := exec.Command("docker", "pull", fmt.Sprintf("%s:%s", image.Name, image.Tag))
	// TODO: Read output from proc using a pipe
	// output:=proc.StdoutPipe()

	outputBytes, err := proc.Output()
	if err != nil {
		exitError := err.(*exec.ExitError)
		if exitError != nil && exitError.Stderr != nil {
			image.logger.Errorf("Output: %s", string(exitError.Stderr))
		}
		image.logger.Error(err)
		return err
	}
	image.logger.Debugf("Output: %s", string(outputBytes))
	return nil
}

func (image *DockerImage) IsRunning() (bool, error) {
	image.logger.Debugf("Checking if image %s %s is running", image.Name, image.RepoDigest)
	proc := exec.Command("docker", "ps", "-a")
	outputBytes, err := proc.Output()
	if err != nil {
		exitError := err.(*exec.ExitError)
		if exitError != nil && exitError.Stderr != nil {
			image.logger.Errorf("Output: %s", string(exitError.Stderr))
		}
		image.logger.Error(err)
	}
	outputString := string(outputBytes)
	image.logger.Debugf("Output: %s", outputString)

	containerId, err := image.getContainerId()
	if err != nil {
		image.logger.Warn("Unable to get containerId.")
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

func (image *DockerImage) Start() error {
	image.logger.Debugf("Starting image %s %s", image.Name, image.RepoDigest)
	args := make([]string, 0)
	args = append(args, "run", "--rm", "-d", "-it")
	args = append(args, fmt.Sprintf("%s@%s", image.Name, image.RepoDigest))
	// TODO: add support for passing in arguments
	args = append(args, "bash")
	proc := exec.Command("docker", args...)
	outputBytes, err := proc.Output()
	if err != nil {
		exitError := err.(*exec.ExitError)
		if exitError != nil && exitError.Stderr != nil {
			image.logger.Errorf("Output: %s", string(exitError.Stderr))
		}
		image.logger.Error(err)
	}
	outputString := string(outputBytes)
	image.logger.Debugf("Output: %s", outputString)
	return nil
}

func (image *DockerImage) Stop() error {
	image.logger.Debugf("Stopping image %s %s", image.Name, image.RepoDigest)

	containerId, err := image.getContainerId()
	if err != nil {
		image.logger.Warn("Unable to get containerId.")
	}

	proc := exec.Command("docker", "stop", containerId)
	outputBytes, err := proc.Output()
	if err != nil {
		image.logger.Warn("Unable to stop image.")
		exitError := err.(*exec.ExitError)
		if exitError != nil && exitError.Stderr != nil {
			image.logger.Errorf("Output: %s", string(exitError.Stderr))
		}
		image.logger.Error(err)
		return err
	}
	outputString := string(outputBytes)
	image.logger.Debugf("Output: %s", outputString)
	return nil
}

func (image *DockerImage) Remove() error {
	image.logger.Debugf("Removing image %s %s", image.Name, image.RepoDigest)
	imageId, err := image.getImageId()
	if err != nil {
		image.logger.Warn("Unable to delete previous image.")
		return err
	}
	proc := exec.Command("docker", "rmi", imageId)
	// TODO: Read output from proc using a pipe
	// output:=proc.StdoutPipe()

	outputBytes, err := proc.Output()
	if err != nil {
		exitError := err.(*exec.ExitError)
		if exitError != nil && exitError.Stderr != nil {
			image.logger.Errorf("Output: %s", string(exitError.Stderr))
		}
		image.logger.Error(err)
		return err
	}
	image.logger.Debugf("Output: %s", string(outputBytes))
	return nil
}

func (image *DockerImage) getContainerId() (string, error) {
	imageId, err := image.getImageId()
	if err != nil {
		image.logger.Warn("Unable to get ImageId.")
		return "", err
	}
	// docker container ls --all --filter=ancestor=e4c58958181a --format "{{.ID}}"
	proc := exec.Command("docker", "container", "ls", "--all", fmt.Sprintf("--filter=ancestor=%s", imageId), "--format", "{{.ID}}")
	outputBytes, err := proc.Output()
	if err != nil {
		image.logger.Warn("Unable to get ContainerId.")
		exitError := err.(*exec.ExitError)
		if exitError != nil && exitError.Stderr != nil {
			image.logger.Errorf("Output: %s", string(exitError.Stderr))
		}
		image.logger.Error(err)
		return "", err
	}

	containerId := strings.TrimSpace(string(outputBytes))
	image.logger.Debugf("ContainerId: %s", containerId)
	return containerId, nil
}

func (image *DockerImage) getImageId() (string, error) {
	proc := exec.Command("docker", "image", "inspect", "--format", "'{{json .Id}}'", fmt.Sprintf("%s@%s", image.Name, image.RepoDigest))
	outputBytes, err := proc.Output()
	if err != nil {
		exitError := err.(*exec.ExitError)
		if exitError != nil && exitError.Stderr != nil {
			image.logger.Errorf("Output: %s", string(exitError.Stderr))
		}
		image.logger.Error(err)
		return "", err
	}
	output := string(outputBytes)
	id := strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(output, "\"", ""), "'", ""))
	image.logger.Debugf("ImageId: %s", id)
	return id, nil
}

func NewDockerImage(name string, tag string, repoDigest string, logger golog.Logger, cancelCtx context.Context, cancelFunc context.CancelFunc) *DockerImage {
	return &DockerImage{
		logger:     logger,
		cancelCtx:  cancelCtx,
		cancelFunc: cancelFunc,
		Name:       name,
		RepoDigest: repoDigest,
		Tag:        tag,
	}
}

type DockerImage struct {
	cancelCtx  context.Context
	cancelFunc context.CancelFunc
	logger     golog.Logger
	Name       string
	RepoDigest string
	Tag        string
}
