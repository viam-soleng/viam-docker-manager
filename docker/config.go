package docker

import (
	"errors"

	"go.viam.com/rdk/utils"
)

type Config struct {
	Attributes  utils.AttributeMap `json:"attributes,omitempty"`
	ImageName   string             `json:"image_name"`
	ImageDigest string             `json:"image_digest"`
	RepoDigest  string             `json:"repo_digest"`
	ImageTag    string             `json:"image_tag"`
	ComposeFile []string           `json:"compose_file"`
}

func (conf *Config) Validate(path string) ([]string, error) {
	if conf.ImageName == "" {
		return nil, errors.New("image_name is required")
	}

	if conf.ImageDigest == "" {
		return nil, errors.New("image_digest is required")
	}

	if conf.RepoDigest == "" {
		return nil, errors.New("repo_digest is required")
	}

	if conf.ImageTag == "" {
		return nil, errors.New("image_tag is required")
	}

	return nil, nil
}
