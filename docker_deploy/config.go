package docker_deploy

import (
	"errors"
	"reflect"
	"strings"

	"go.viam.com/rdk/utils"
)

var ErrComposeAndRunOptionsSet = errors.New("only one of run_options or compose_options can be set")
var ErrImageNameRequired = errors.New("image_name is required")
var ErrRepoDigestRequired = errors.New("repo_digest is required")
var ErrComposeFileRequired = errors.New("compose_file is required")
var ErrComposeRepoDigestRequired = errors.New("repo_digest is required in compose_file")
var ErrUsernameIsRequired = errors.New("credentials.username is required")
var ErrPasswordIsRequired = errors.New("credentials.password is required")
var ErrAutoRemoveType = errors.New("host_options 'AutoRemove' parameter must be a boolean")
var ErrBindType = rrors.New("host_options 'Binds' parameter must include a non-empty string")
var ErrNetworkModeType = errors.New("host_options 'Network Mode' parameter must include a non-empty string")

type Config struct {
	Attributes     utils.AttributeMap `json:"attributes,omitempty"`
	RunOptions     *RunOptions        `json:"run_options"`
	ComposeOptions *ComposeOptions    `json:"compose_options"`
	ImageName      string             `json:"image_name"`
	RepoDigest     string             `json:"repo_digest"`
	RunOnce        bool               `json:"run_once"`
	DownloadOnly   bool               `json:"download_only"`
	Credentials    *Credentials       `json:"credentials"`
}

// This is for docker compose based configs
type ComposeOptions struct {
	ComposeFile []string `json:"compose_file"`
}

type RunOptions struct {
	Env            []string               `json:"env"`
	EntryPointArgs []string               `json:"entry_point_args"`
	Options        map[string]interface{} `json:"options"`
	HostOptions    map[string]interface{} `json:"host_options"`
}

type Credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (conf *Config) HasChanged(newConf *Config) bool {
	if conf.ImageName != newConf.ImageName ||
		conf.RepoDigest != newConf.RepoDigest {
		return true
	}
	if conf.RunOptions != nil && newConf.RunOptions != nil {
		return !stringSliceEqual(conf.RunOptions.Env, newConf.RunOptions.Env) ||
			!stringSliceEqual(conf.RunOptions.EntryPointArgs, newConf.RunOptions.EntryPointArgs) ||
			!mapsEqual(conf.RunOptions.Options, newConf.RunOptions.Options) ||
			!mapsEqual(conf.RunOptions.HostOptions, newConf.RunOptions.HostOptions) ||
			(conf.Credentials != nil && newConf.Credentials != nil && (conf.Credentials.Username != newConf.Credentials.Username || conf.Credentials.Password != newConf.Credentials.Password))
	} else if conf.ComposeOptions != nil && newConf.ComposeOptions != nil {
		return !stringSliceEqual(conf.ComposeOptions.ComposeFile, newConf.ComposeOptions.ComposeFile) ||
			(conf.Credentials != nil && newConf.Credentials != nil && (conf.Credentials.Username != newConf.Credentials.Username || conf.Credentials.Password != newConf.Credentials.Password))
	}
	return false
}

func (conf *Config) Validate(path string) ([]string, error) {
	var validationErrors []error
	if conf.RunOptions != nil && conf.ComposeOptions != nil {
		return nil, ErrComposeAndRunOptionsSet
	}

	if conf.ImageName == "" {
		validationErrors = append(validationErrors, ErrImageNameRequired)
	}

	if conf.RepoDigest == "" {
		validationErrors = append(validationErrors, ErrRepoDigestRequired)
	}

	if conf.ComposeOptions != nil {
		if conf.ComposeOptions.ComposeFile == nil {
			validationErrors = append(validationErrors, ErrComposeFileRequired)
		}

		// We need to make sure that the repo digest is contained in the compose file, otherwise running the compose file will pull the latest image
		containsRepoDigest := false
		for _, line := range conf.ComposeOptions.ComposeFile {
			if strings.Contains(line, conf.RepoDigest) {
				containsRepoDigest = true
			}
		}
		if !containsRepoDigest {
			return nil, ErrComposeRepoDigestRequired
		}
	}

	if conf.RunOptions != nil {
		host_opts := conf.RunOptions.HostOptions
		if host_opts != nil && len(host_opts) != 0 {
			if bind, ok := host_opts["Binds"].(string); !ok || bind == "" {
				validationErrors = append(validationErrors, ErrBindType)
			}
			if networkMode, ok := host_opts["NetworkMode"].(string); !ok || networkMode == "" {
				validationErrors = append(validationErrors, ErrNetworkModeType)
			}
			if _, ok := host_opts["AutoRemove"].(bool); !ok {
				validationErrors = append(validationErrors, ErrAutoRemoveType)
			}
		}
	}

	if conf.Credentials != nil {
		if conf.Credentials.Username == "" {
			validationErrors = append(validationErrors, ErrUsernameIsRequired)
		}
		if conf.Credentials.Password == "" {
			validationErrors = append(validationErrors, ErrPasswordIsRequired)
		}
	}

	return nil, errors.Join(validationErrors...)
}

// StringSliceEqual checks if two string slices are equal
func stringSliceEqual(a, b []string) bool {
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

// mapsEqual checks if two maps are equal
func mapsEqual(a, b map[string]interface{}) bool {
	return reflect.DeepEqual(a, b)
}
