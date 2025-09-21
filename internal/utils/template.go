// Template processing using gomplate
package utils

import (
	"encoding/json"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/bedag/subst/internal/wrapper"
	"sigs.k8s.io/yaml"
)

// Template processes templates using gomplate
func Template(data []byte, values map[interface{}]interface{}) (map[interface{}]interface{}, error) {
	return TemplateWithEngine(data, values, "")
}

// TemplateWithEngine processes templates using gomplate with environment regex
func TemplateWithEngine(data []byte, values map[interface{}]interface{}, envRegex string) (map[interface{}]interface{}, error) {
	// Convert values to string map for gomplate
	envData := ToMap(values)
	
	// Process with gomplate
	processed, err := wrapper.ProcessGomplateTemplate(data, envData, envRegex)
	if err != nil {
		return nil, err
	}
	
	return ParseYAML(processed)
}

// Utility functions for data conversion (keeping these as they're used elsewhere)

// toYAML takes an interface, marshals it to yaml, and returns a string. It will
// always return a string, even on marshal error (empty string).
func toYAML(v interface{}) string {
	data, err := yaml.Marshal(v)
	if err != nil {
		// Swallow errors inside of a template.
		return ""
	}
	return strings.TrimSuffix(string(data), "\n")
}

// fromYAML converts a YAML document into a map[string]interface{}.
func fromYAML(str string) map[string]interface{} {
	m := map[string]interface{}{}

	if err := yaml.Unmarshal([]byte(str), &m); err != nil {
		m["Error"] = err.Error()
	}
	return m
}

// fromYAMLArray converts a YAML array into a []interface{}.
func fromYAMLArray(str string) []interface{} {
	a := []interface{}{}

	if err := yaml.Unmarshal([]byte(str), &a); err != nil {
		a = []interface{}{err.Error()}
	}
	return a
}

// toTOML takes an interface, marshals it to toml, and returns a string.
func toTOML(v interface{}) string {
	var buf strings.Builder
	e := toml.NewEncoder(&buf)
	err := e.Encode(v)
	if err != nil {
		return err.Error()
	}
	return buf.String()
}

// toJSON takes an interface, marshals it to json, and returns a string.
func toJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		// Swallow errors inside of a template.
		return ""
	}
	return string(data)
}

// fromJSON converts a JSON document into a map[string]interface{}.
func fromJSON(str string) map[string]interface{} {
	m := make(map[string]interface{})

	if err := json.Unmarshal([]byte(str), &m); err != nil {
		m["Error"] = err.Error()
	}
	return m
}

// fromJSONArray converts a JSON array into a []interface{}.
func fromJSONArray(str string) []interface{} {
	a := []interface{}{}

	if err := json.Unmarshal([]byte(str), &a); err != nil {
		a = []interface{}{err.Error()}
	}
	return a
}
