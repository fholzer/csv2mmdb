package convert

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DatabaseType  string         `yaml:"databaseType"`
	RecordSize    uint8          `yaml:"recordSize"`
	UseValueCache bool           `yaml:"useValueCache"`
	Fields        []*FieldConfig `yaml:"fields"`
}

func (c *Config) Validate() error {
	for _, f := range c.Fields {
		if err := f.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type FieldConfig struct {
	Name           string            `yaml:"name"`
	Target         string            `yaml:"target"`
	Type           string            `yaml:"type"`
	Capitalization string            `yaml:"capitalization"`
	Translate      map[string]string `yaml:"translate"`
	IgnoreEmpty    bool              `yaml:"ignoreEmpty"`
	Critical       bool              `yaml:"critical"`
	OmitZeroValue  bool              `yaml:"omitZeroValue"`
	FieldMapper    FieldMapper
}

func (f *FieldConfig) Validate() error {
	if f.Type == "" {
		f.Type = "string"
	}
	switch f.Type {
	case "string":
	case "int32":
	case "uint32":
	case "int64":
	case "uint64":
	case "boolean":
	case "float32":
	default:
		return fmt.Errorf("unknown field type '%s' for field '%s'", f.Type, f.Name)
	}

	switch f.Capitalization {
	case "":
	case "lower":
	case "upper":
	case "title":
	default:
		return fmt.Errorf("unknown capitalization mode '%s' for fiel '%s'", f.Capitalization, f.Name)
	}
	return nil
}

func NewConfig(filePath string) (*Config, error) {
	// Create config structure
	config := &Config{}

	// Open config file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Init new YAML decode
	d := yaml.NewDecoder(file)
	d.KnownFields(true)

	// Start YAML decoding from file
	if err := d.Decode(&config); err != nil {
		return nil, err
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	// fmt.Printf("config: ")
	// spew.Dump(config)
	return config, nil
}
