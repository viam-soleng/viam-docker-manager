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

func (image *DockerImage) IsRunning() bool {
	image.logger.Debugf("Checking if image %s %s is running", image.Name, image.RepoDigest)
	return false
}

func (image *DockerImage) Start() error {
	image.logger.Debugf("Starting image %s %s", image.Name, image.RepoDigest)
	return nil
}

func (image *DockerImage) Stop() error {
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
