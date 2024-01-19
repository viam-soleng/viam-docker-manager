package docker_deploy

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/compose-spec/compose-go/loader"
	compose_types "github.com/compose-spec/compose-go/types"
	docker_types "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"go.viam.com/rdk/logging"
)

var ErrImageNotFound = errors.New("image not found")
var ErrContainerNotFound = errors.New("container not found")

type DockerManager interface {
	ListContainers() ([]DockerContainerDetails, error)
	CreateContainer(imageName string, repoDigest string, entry_point_args []string, options []string, logger logging.Logger, cancelCtx context.Context, cancelFunc context.CancelFunc) (DockerContainer, error)
	CreateComposeContainers(imageName string, repoDigest string, composeFile []string, logger logging.Logger, cancelCtx context.Context, cancelFunc context.CancelFunc) ([]DockerContainer, error)

	ListImages() ([]DockerImageDetails, error)
	GetImageDetails(imageId string) (*DockerImageDetails, error)
	GetContainer(containerId string) (*DockerContainerDetails, error)
	GetContainerImageDigest(containerId string) (string, error)
	GetContainersRunningImage(imageDigest string) ([]DockerContainerDetails, error)

	PullImage(imageName string, repoDigest string) error
	ImageExists(repoDigest string) (bool, error)
	RemoveImageByImageId(imageId string) error
	RemoveImageByRepoDigest(repoDigest string) error

	StartContainer(containerId string) error
	StopContainer(containerId string) error
	RemoveContainer(containerId string) error
}

type LocalDockerManager struct {
	logger       logging.Logger
	dockerClient *client.Client
	username     string
	password     string
}

type DockerImageDetails struct {
	Repository string
	Tag        string
	RepoDigest string
	ImageID    string
	Created    time.Time
	Size       int64
}

type DockerContainerDetails struct {
	ContainerID string
	ImageID     string
	Command     string
	Created     time.Time
	Status      string
	Ports       string
	Names       string
}

func NewLocalDockerManagerWithAuth(username string, password string, logger logging.Logger) (DockerManager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	return &LocalDockerManager{logger: logger, dockerClient: cli, username: username, password: password}, err
}

func NewLocalDockerManager(logger logging.Logger) (DockerManager, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	return &LocalDockerManager{logger: logger, dockerClient: cli}, err
}

func (dm *LocalDockerManager) ListImages() ([]DockerImageDetails, error) {
	imgs, err := dm.dockerClient.ImageList(context.Background(), docker_types.ImageListOptions{All: true})
	if err != nil {
		return nil, err
	}
	var images []DockerImageDetails

	for _, image := range imgs {
		repoDigest := image.RepoDigests[0]
		s := strings.Split(repoDigest, "@")

		tag := ""
		if len(image.RepoTags) > 0 {
			tag = image.RepoTags[0]
		}

		images = append(images, DockerImageDetails{
			Repository: s[0],
			Tag:        tag,
			RepoDigest: repoDigest,
			ImageID:    image.ID,
			Created:    time.Unix(image.Created, 0),
			Size:       image.Size,
		})
	}

	return images, nil
}

func (dm *LocalDockerManager) RemoveImageByImageId(imageId string) error {
	resp, err := dm.dockerClient.ImageRemove(context.Background(), imageId, docker_types.ImageRemoveOptions{Force: true})
	if err != nil {
		return err
	}
	for _, r := range resp {
		if r.Deleted != "" {
			dm.logger.Infof("Image %s has been removed", r.Deleted)
		}
	}
	return nil
}

func (dm *LocalDockerManager) RemoveImageByRepoDigest(repoDigest string) error {
	images, err := dm.ListImages()
	if err != nil {
		return err
	}
	var imageId string
	for _, image := range images {
		if image.RepoDigest == repoDigest {
			imageId = image.ImageID
		}
	}
	return dm.RemoveImageByImageId(imageId)
}

func (dm *LocalDockerManager) ListContainers() ([]DockerContainerDetails, error) {
	c, err := dm.dockerClient.ContainerList(context.Background(), docker_types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}

	var containers []DockerContainerDetails

	for _, container := range c {
		var ports strings.Builder
		for _, port := range container.Ports {
			ports.WriteString(fmt.Sprintf("%d:%d, ", port.PublicPort, port.PrivatePort))
		}
		containers = append(containers, DockerContainerDetails{
			ContainerID: container.ID,
			ImageID:     container.ImageID,
			Command:     container.Command,
			Created:     time.Unix(container.Created, 0),
			Status:      container.Status,
			Ports:       ports.String(),
			Names:       container.Names[0],
		})
	}

	return containers, nil
}

func (dm *LocalDockerManager) GetImageDetails(imageId string) (*DockerImageDetails, error) {
	images, err := dm.ListImages()
	if err != nil {
		return nil, err
	}

	for _, image := range images {
		if image.ImageID == imageId {
			return &image, nil
		}
	}
	return nil, errors.New("image not found")
}

func (dm *LocalDockerManager) GetContainer(containerId string) (*DockerContainerDetails, error) {
	containers, err := dm.ListContainers()
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		if container.ContainerID == containerId {
			return &container, nil
		}
	}
	return nil, errors.New("container not found")
}

func (dm *LocalDockerManager) GetContainerImageDigest(containerId string) (string, error) {
	var container *DockerContainerDetails
	containers, err := dm.ListContainers()
	if err != nil {
		return "", err
	}
	for _, c := range containers {
		if c.ContainerID == containerId {
			container = &c
		}
	}
	if container == nil {
		return "", ErrContainerNotFound
	}

	images, err := dm.ListImages()
	if err != nil {
		return "", err
	}
	for _, image := range images {
		if image.ImageID == container.ImageID {
			return image.RepoDigest, nil
		}
	}

	return "", ErrImageNotFound
}

func (dm *LocalDockerManager) GetContainersRunningImage(repoDigest string) ([]DockerContainerDetails, error) {
	containers, err := dm.ListContainers()
	if err != nil {
		return nil, err
	}

	var containersRunningImage []DockerContainerDetails
	for _, container := range containers {
		digest, err := dm.GetContainerImageDigest(container.ContainerID)
		if err != nil {
			continue
		}
		if digest == repoDigest {
			containersRunningImage = append(containersRunningImage, container)
		}
	}

	return containersRunningImage, nil
}

func (dm *LocalDockerManager) PullImage(imageName string, repoDigest string) error {
	dm.logger.Debugf("Pulling image %s %s", imageName, repoDigest)
	imagePullOptions := docker_types.ImagePullOptions{}

	if dm.username != "" && dm.password != "" {
		imagePullOptions = docker_types.ImagePullOptions{
			RegistryAuth: base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf(`{"username":"%s","password":"%s"}`, dm.username, dm.password))),
		}
	}
	rc, err := dm.dockerClient.ImagePull(context.Background(), fmt.Sprintf("%s@%s", imageName, repoDigest), imagePullOptions)
	if err != nil {
		return err
	}
	defer rc.Close()

	// ...
	decoder := json.NewDecoder(rc)
	for {
		var message map[string]interface{}
		if err := decoder.Decode(&message); err != nil {
			if err == io.EOF {
				break
			}
			dm.logger.Warn(err)
		}

		// process message
		dm.logger.Debugf("%v", message)
	}

	return nil
}

func (dm *LocalDockerManager) CreateContainer(imageName string, repoDigest string, entry_point_args []string, options []string, logger logging.Logger, cancelCtx context.Context, cancelFunc context.CancelFunc) (DockerContainer, error) {
	config := &container.Config{
		Image: imageName,
		Cmd:   entry_point_args,
	}

	resp, err := dm.dockerClient.ContainerCreate(cancelCtx, config, nil, nil, nil, "")
	if err != nil {
		return nil, err
	}

	for _, w := range resp.Warnings {
		logger.Warnf("Create container warning: %s", w)
	}

	c := NewDockerContainer(dm.dockerClient, resp.ID, imageName, repoDigest, logger, cancelCtx, cancelFunc)
	return c, nil
}

func (dm *LocalDockerManager) CreateComposeContainers(imageName string, repoDigest string, composeFile []string, logger logging.Logger, cancelCtx context.Context, cancelFunc context.CancelFunc) ([]DockerContainer, error) {
	ctx := context.Background()
	sanitizedImageName := strings.Replace(imageName, "/", "-", -1)
	composeFileName := fmt.Sprintf("%s/%s-%s.yml", os.TempDir(), "docker-compose", sanitizedImageName)
	b := make([]byte, 0)
	for _, line := range composeFile {
		s := []byte(fmt.Sprintln(line))
		b = append(b, []byte(s)...)
	}

	yaml, err := loader.ParseYAML(b)
	if err != nil {
		return nil, err
	}

	project, err := loader.Load(compose_types.ConfigDetails{
		WorkingDir:  ".",
		ConfigFiles: []compose_types.ConfigFile{{Config: yaml, Filename: composeFileName}},
		Environment: map[string]string{},
	})
	if err != nil {
		return nil, err
	}

	containers := make([]DockerContainer, 0, len(project.Services))
	for _, service := range project.Services {
		env := make([]string, 0, len(service.Environment))
		for k, v := range service.Environment {
			env = append(env, fmt.Sprintf("%s=%v", k, v))
		}

		exposedPorts := nat.PortSet{}
		for _, port := range service.Ports {
			natPort, err := nat.NewPort(port.Protocol, fmt.Sprint(port.Target))
			if err != nil {
				return nil, err
			}
			exposedPorts[natPort] = struct{}{}
		}
		config := &container.Config{
			Image:        service.Image,
			Env:          env,
			ExposedPorts: exposedPorts,
		}

		resp, err := dm.dockerClient.ContainerCreate(ctx, config, nil, nil, nil, service.Name)
		if err != nil {
			return nil, err
		}

		dm.logger.Infof("Container %s has been created", resp.ID)
		containers = append(containers, NewDockerContainer(dm.dockerClient, resp.ID, imageName, repoDigest, logger, cancelCtx, cancelFunc))
	}
	return containers, nil
}

func (dm *LocalDockerManager) ImageExists(repoDigest string) (bool, error) {
	images, err := dm.ListImages()
	if err != nil {
		return false, err
	}

	for _, image := range images {
		if image.RepoDigest == repoDigest {
			return true, nil
		}
	}
	return false, nil
}

func (dm *LocalDockerManager) StartContainer(containerId string) error {
	return dm.dockerClient.ContainerStart(context.Background(), containerId, docker_types.ContainerStartOptions{})
}

func (dm *LocalDockerManager) StopContainer(containerId string) error {
	return dm.dockerClient.ContainerStop(context.Background(), containerId, container.StopOptions{})
}

func (dm *LocalDockerManager) RemoveContainer(containerId string) error {
	return dm.dockerClient.ContainerRemove(context.Background(), containerId, docker_types.ContainerRemoveOptions{Force: true})
}
