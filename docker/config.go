package docker

import "go.viam.com/rdk/utils"

type Config struct {
	Attributes  utils.AttributeMap `json:"attributes,omitempty"`
	ImageName   string             `json:"image_name"`
	ImageDigest string             `json:"image_hash"`
	RepoDigest  string             `json:"repo_digest"`
	ImageTag    string             `json:"image_tag"`
}

func (conf *Config) Validate(path string) ([]string, error) {
	return nil, nil
}
