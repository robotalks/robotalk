package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/easeway/langx.go/mapper"
	yaml "gopkg.in/yaml.v2"
)

// Config is abstraction of configuration
type Config interface {
	// As converts the configuration to specified type
	As(out interface{}) error
}

// MapConfig implements Config backed by a map
type MapConfig struct {
	Map map[string]interface{}
}

// NewMapConfig creates a MapConfig
func NewMapConfig() *MapConfig {
	return &MapConfig{Map: make(map[string]interface{})}
}

// As implements Config
func (c *MapConfig) As(out interface{}) error {
	if c.Map == nil {
		return nil
	}
	return mapper.Map(out, c.Map)
}

// LoadFile loads config from JSON/YAML
func (c *MapConfig) LoadFile(fn string) error {
	if fn == "-" || fn == "" {
		return c.Load(os.Stdin)
	}
	content, err := ioutil.ReadFile(fn)
	if err == nil {
		err = c.Load(bytes.NewBuffer(content))
	}
	return err
}

// Load loads config from a stream
func (c *MapConfig) Load(stream io.Reader) error {
	content, err := ioutil.ReadAll(stream)
	if err != nil {
		return err
	}
	if c.Map == nil {
		c.Map = make(map[string]interface{})
	}
	if bytes.HasPrefix(bytes.TrimSpace(content), []byte{'{'}) {
		err = json.Unmarshal(content, c.Map)
	} else {
		err = yaml.Unmarshal(content, c.Map)
		if err == nil {
			if m, ok := normalizeMap(c.Map).(map[string]interface{}); ok {
				c.Map = m
			} else {
				err = fmt.Errorf("invalid forrmat")
			}
		}
	}
	return err
}

func normalizeMap(val interface{}) interface{} {
	switch v := val.(type) {
	case []interface{}:
		for n, item := range v {
			v[n] = normalizeMap(item)
		}
	case map[interface{}]interface{}:
		m := make(map[string]interface{})
		for key, value := range v {
			m[fmt.Sprintf("%v", key)] = normalizeMap(value)
		}
		val = m
	case map[string]interface{}:
		for key, value := range v {
			v[key] = normalizeMap(value)
		}
	}
	return val
}
