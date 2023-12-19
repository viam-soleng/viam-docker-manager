package docker

import (
	"errors"
	"strings"

	"go.viam.com/rdk/utils"
)

type Config struct {
	Attributes utils.AttributeMap `json:"attributes,omitempty"`
	ImageName  string             `json:"image_name"`
	RepoDigest string             `json:"repo_digest"`

	// This is for docker compose based configs
	ComposeFile []string `json:"compose_file"`

	// This is for docker run based configs
	EntryPointArgs []string `json:"entry_point_args"`
	Options        []string `json:"options"`
	RunOnce        bool     `json:"run_once"`
	DownloadOnly   bool     `json:"download_only"`
}

func (conf *Config) Validate(path string) ([]string, error) {
	if conf.ImageName == "" {
		return nil, errors.New("image_name is required")
	}

	if conf.RepoDigest == "" {
		return nil, errors.New("repo_digest is required")
	}

	// We need to make sure that the repo digest is contained in the compose file, otherwise running the compose file will pull the latest image
	if conf.ComposeFile != nil {
		containsRepoDigest := false
		for _, line := range conf.ComposeFile {
			if strings.Contains(line, conf.RepoDigest) {
				containsRepoDigest = true
			}
		}
		if !containsRepoDigest {
			return nil, errors.New("repo_digest must be in compose_file")
		}
	}

	return nil, nil
}
