package yamlparser

import (
	"gopkg.in/yaml.v2"
	"os"
	"path/filepath"
)

type Config struct {
	Service []struct {
		Name       string `yaml:"name"`
		UrlPrefix  string `yaml:"url_prefix"`
		ServerPool []struct {
			ContainerName string `yaml:"container_name"`
			Port          uint16 `yaml:"port"`
		} `yaml:"server_pool"`
	} `yaml:"service"`
}

func GetConfig(path string) (Config, error) {
	filename, _ := filepath.Abs(path)
	yamlFile, err := os.ReadFile(filename)

	if err != nil {
		panic(err)
	}

	var config Config

	err = yaml.UnmarshalStrict(yamlFile, &config)
	if err != nil {
		return config, err
	}

	return config, nil
}
