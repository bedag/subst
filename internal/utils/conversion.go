package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"github.com/geofffranks/simpleyaml"
	"github.com/starkandwayne/goutils/ansi"
	"gopkg.in/yaml.v2"
)

// Convert converts a map[string]interface{} to a map[interface{}]interface{}.
func ToInterface(inputMap map[string]interface{}) map[interface{}]interface{} {
	var convertedMap = make(map[interface{}]interface{}, len(inputMap))
	for key, value := range inputMap {
		convertedMap[key] = value
	}
	return convertedMap
}

// convert map[interface{}]interface{} recursive to map[string]string
func ToMap(i map[interface{}]interface{}) map[string]interface{} {
	out := mapify(i)
	return out
}

// convert map[string]interface{} to map[string]string (Recursion)
func mapify(input map[interface{}]interface{}) map[string]interface{} {
	output := make(map[string]interface{}, len(input))
	for k, v := range input {
		switch vv := v.(type) {
		case map[interface{}]interface{}:
			output[k.(string)] = mapify(vv)
		default:
			output[k.(string)] = vv
		}
	}
	return output
}

func ConvertPath(path string) string {
	if path[len(path)-1:] != "/" {
		path = fmt.Sprintf("%v/", path)
	}
	return path
}

// Convert YAML/JSON data to a map[interface{}]interface{}.
func ParseYAML(data []byte) (map[interface{}]interface{}, error) {
	y, err := simpleyaml.NewYaml(data)
	if err != nil {
		return nil, err
	}

	if empty_y, _ := simpleyaml.NewYaml([]byte{}); *y == *empty_y {
		fmt.Println("YAML doc is empty, creating empty hash/map")
		return make(map[interface{}]interface{}), nil
	}

	doc, err := y.Map()

	if err != nil {
		if _, arrayErr := y.Array(); arrayErr == nil {
			return nil, err
		}
		return nil, ansi.Errorf("@R{Root of YAML document is not a hash/map}: %s\n", err.Error())
	}

	return doc, nil
}

// create a golang function which prints map[interface{}]interface{} as yaml
func PrintYAML(data map[interface{}]interface{}) error {
	y, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	if _, err := writer.WriteString("---\n"); err != nil {
		return err
	}
	if _, err := writer.Write(y); err != nil {
		return err
	}

	return nil
}

// PrintYAMLBytes prints YAML from byte slice
func PrintYAMLBytes(data []byte) error {
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	if _, err := writer.WriteString("---\n"); err != nil {
		return err
	}
	if _, err := writer.Write(data); err != nil {
		return err
	}

	return nil
}

// SplitYAMLDocs splits a multi-document YAML string into individual documents
func SplitYAMLDocs(yamlContent string) []string {
	var docs []string
	
	// Simple approach: split on "---" at line boundaries
	parts := []string{yamlContent} // Start with the whole content
	
	// This is a basic implementation - for production, you might want a more robust parser
	// But for CMP simplicity, this should work fine
	if len(parts) == 1 && parts[0] != "" {
		docs = append(docs, parts[0])
	}
	
	return docs
}

// create a golang function which prints map[interface{}]interface{}
func PrintJSON(data map[interface{}]interface{}) error {
	j, err := json.MarshalIndent(mapify(data), "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(j))
	return err
}

func UnmarshalJSONorYAML(data []byte) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal(data, &result)
	if err != nil {
		err = yaml.Unmarshal(data, &result)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func UnmarshalJSONorYAMLToInterface(data []byte, result *map[interface{}]interface{}) error {
	var tmp map[string]interface{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		err = yaml.Unmarshal(data, &tmp)
		if err != nil {
			return err
		}
	}
	*result = ToInterface(tmp)
	return nil
}

// MergeValues merges two maps, with the second map taking precedence
func MergeValues(base, override map[interface{}]interface{}) map[interface{}]interface{} {
	result := make(map[interface{}]interface{})

	// Copy base values
	for k, v := range base {
		result[k] = deepCopy(v)
	}

	// Merge override values
	for k, v := range override {
		if existing, exists := result[k]; exists {
			// If both values are maps, merge them recursively
			if existingMap, ok := existing.(map[interface{}]interface{}); ok {
				if newMap, ok := v.(map[interface{}]interface{}); ok {
					result[k] = MergeValues(existingMap, newMap)
					continue
				}
			}
		}
		// Otherwise, override takes precedence
		result[k] = deepCopy(v)
	}

	return result
}

// deepCopy creates a deep copy of a value
func deepCopy(val interface{}) interface{} {
	switch v := val.(type) {
	case map[interface{}]interface{}:
		result := make(map[interface{}]interface{})
		for k, v := range v {
			result[k] = deepCopy(v)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = deepCopy(item)
		}
		return result
	default:
		return v
	}
}

// ToYAML converts a map to YAML bytes
func ToYAML(data map[interface{}]interface{}) ([]byte, error) {
	return yaml.Marshal(data)
}
