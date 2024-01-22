package docker_deploy

import (
	"errors"
	"strings"

	"go.viam.com/rdk/utils"
)

var ErrImageNameRequired = errors.New("image_name is required")
var ErrRepoDigestRequired = errors.New("repo_digest is required")

type Config struct {
	Attributes     utils.AttributeMap `json:"attributes,omitempty"`
	RunOptions     *RunOptions        `json:"run_options"`
	ComposeOptions *ComposeOptions    `json:"compose_options"`
	RunOnce        bool               `json:"run_once"`
	DownloadOnly   bool               `json:"download_only"`
	Credentials    *Credentials       `json:"credentials"`
}

// This is for docker compose based configs
type ComposeOptions struct {
	ImageName   string   `json:"image_name"`
	RepoDigest  string   `json:"repo_digest"`
	ComposeFile []string `json:"compose_file"`
}

type RunOptions struct {
	ImageName      string   `json:"image_name"`
	RepoDigest     string   `json:"repo_digest"`
	Env            []string `json:"env"`
	EntryPointArgs []string `json:"entry_point_args"`
	Options        []string `json:"options"`
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (conf *Config) HasChanged(newConf *Config) bool {
	if conf.RunOptions != nil && newConf.RunOptions != nil {
		return conf.RunOptions.ImageName != newConf.RunOptions.ImageName ||
			conf.RunOptions.RepoDigest != newConf.RunOptions.RepoDigest ||
			!StringSliceEqual(conf.RunOptions.Env, newConf.RunOptions.Env) ||
			!StringSliceEqual(conf.RunOptions.EntryPointArgs, newConf.RunOptions.EntryPointArgs) ||
			!StringSliceEqual(conf.RunOptions.Options, newConf.RunOptions.Options) ||
			(conf.Credentials != nil && newConf.Credentials != nil && (conf.Credentials.Username != newConf.Credentials.Username || conf.Credentials.Password != newConf.Credentials.Password))
	} else if conf.ComposeOptions != nil && newConf.ComposeOptions != nil {
		return conf.ComposeOptions.ImageName != newConf.ComposeOptions.ImageName ||
			conf.ComposeOptions.RepoDigest != newConf.ComposeOptions.RepoDigest ||
			!StringSliceEqual(conf.ComposeOptions.ComposeFile, newConf.ComposeOptions.ComposeFile) ||
			(conf.Credentials != nil && newConf.Credentials != nil && (conf.Credentials.Username != newConf.Credentials.Username || conf.Credentials.Password != newConf.Credentials.Password))
	}
	return false
}

func (conf *Config) Validate(path string) ([]string, error) {
	var validationErrors []error
	if conf.RunOptions != nil && conf.ComposeOptions != nil {
		return nil, errors.New("only one of run_options or compose_options can be set")
	}

	if conf.RunOptions != nil {
		if conf.RunOptions.ImageName == "" {
			validationErrors = append(validationErrors, errors.New("image_name is required"))
		}

		if conf.RunOptions.RepoDigest == "" {
			validationErrors = append(validationErrors, errors.New("repo_digest is required"))
		}
	}

	if conf.ComposeOptions != nil {
		if conf.ComposeOptions.ImageName == "" {
			validationErrors = append(validationErrors, errors.New("image_name is required"))
		}

		if conf.ComposeOptions.RepoDigest == "" {
			validationErrors = append(validationErrors, errors.New("repo_digest is required"))
		}

		if conf.ComposeOptions.ComposeFile == nil {
			validationErrors = append(validationErrors, errors.New("compose_file is required"))
		}

		// We need to make sure that the repo digest is contained in the compose file, otherwise running the compose file will pull the latest image
		containsRepoDigest := false
		for _, line := range conf.ComposeOptions.ComposeFile {
			if strings.Contains(line, conf.ComposeOptions.RepoDigest) {
				containsRepoDigest = true
			}
		}
		if !containsRepoDigest {
			return nil, errors.New("repo_digest must be in compose_file")
		}
	}

	return nil, errors.Join(validationErrors...)
}

func (conf *Config) GetImageName() (string, error) {
	if conf.RunOptions != nil {
		return conf.RunOptions.ImageName, nil
	} else if conf.ComposeOptions != nil {
		return conf.ComposeOptions.ImageName, nil
	}
	return "", ErrImageNameRequired
}

func (conf *Config) GetRepoDigest() (string, error) {
	if conf.RunOptions != nil {
		return conf.RunOptions.RepoDigest, nil
	} else if conf.ComposeOptions != nil {
		return conf.ComposeOptions.RepoDigest, nil
	}
	return "", ErrRepoDigestRequired
}

// StringSliceEqual checks if two string slices are equal
func StringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if v != b[i] {
			return false
		}
	}
	return true
}
