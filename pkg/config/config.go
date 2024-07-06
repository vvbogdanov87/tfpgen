package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"

	"k8s.io/apimachinery/pkg/util/yaml"
)

type Config struct {
	// Name is the provider name.
	Name string `yaml:"name"`
	// Address is the provider address for the Terraform registry.
	Address string `yaml:"address"`
	// ModuleName is the name of the Go module.
	ModuleName string `yaml:"moduleName"`
	// SchemasDir is the directory containing the CRD schemas.
	SchemasDir string `yaml:"schemasDir"`
	// OutputDir is the directory to write the generated provider code.
	OutputDir string `yaml:"outputDir"`

	// The directory of the configuration file.
	// All paths in the configuration file are relative to this directory.
	baseDir string
}

func NewConfig(filename string) (*Config, error) {
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

	err = config.setDefaults(filepath.Dir(filename))
	if err != nil {
		return nil, fmt.Errorf("failed to set defaults: %w", err)
	}

	return config, nil
}

func (c *Config) setDefaults(baseDir string) error {
	if filepath.IsAbs(baseDir) {
		c.baseDir = baseDir
	} else {
		baseDir, err := filepath.Abs(baseDir)
		if err != nil {
			return fmt.Errorf("failed to get absolute path: %w", err)
		}
		c.baseDir = baseDir
	}

	c.SchemasDir = filepath.Join(c.baseDir, c.SchemasDir)
	c.OutputDir = filepath.Join(c.baseDir, c.OutputDir)

	return nil
}
