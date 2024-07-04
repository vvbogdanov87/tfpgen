package config

import (
	"bufio"
	"fmt"
	"os"

	"k8s.io/apimachinery/pkg/util/yaml"
)

type Config struct {
	// Name is the provider name.
	Name string `yaml:"name"`
	// Address is the provider address for the Terraform registry.
	Address string `yaml:"address"`
	// SchemasDir is the directory containing the CRD schemas.
	SchemasDir string `yaml:"schemasDir"`
	// OutputDir is the directory to write the generated provider code.
	OutputDir string `yaml:"outputDir"`
}

func ReadConfig(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer f.Close()

	bufr := bufio.NewReader(f)
	yamlReader := yaml.NewYAMLReader(bufr)
	data, err := yamlReader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read yaml from file %s: %w", filename, err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal yaml from file %s: %w", filename, err)
	}

	return config, nil
}
