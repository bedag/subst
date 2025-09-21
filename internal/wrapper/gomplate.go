package wrapper

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// IsGomplateTemplate checks if content contains gomplate-like syntax
func IsGomplateTemplate(content []byte) bool {
	contentStr := string(content)
	
	// Check for common gomplate patterns
	gomplatePatterns := []string{
		"{{ env.Getenv",
		"{{ .Env.",
		"{{ datasource",
		"{{ include",
		"{{ tmpl.Exec",
		"{{ aws.",
		"{{ vault.",
		"{{ consul.",
		"{{ .Values.",
	}
	
	for _, pattern := range gomplatePatterns {
		if strings.Contains(contentStr, pattern) {
			return true
		}
	}
	
	// Check for template delimiters with function calls
	if strings.Contains(contentStr, "{{") && strings.Contains(contentStr, "}}") {
		// Look for function-like patterns that are common in gomplate
		functionPatterns := []string{
			"env.",
			"file.",
			"conv.",
			"strings.",
			"json.",
			"yaml.",
			"base64.",
			"crypto.",
			"math.",
			"time.",
			"uuid.",
		}
		
		for _, funcPattern := range functionPatterns {
			if strings.Contains(contentStr, funcPattern) {
				return true
			}
		}
	}
	
	return false
}

// ProcessGomplateTemplate processes templates using gomplate binary if available,
// otherwise falls back to a simplified template processor
func ProcessGomplateTemplate(templateContent []byte, envData map[string]interface{}, envRegex string) ([]byte, error) {
	// Try to use gomplate binary first
	if gomplateAvailable() {
		return processWithGomplateBinary(templateContent, envData)
	}
	
	// Fallback to go text/template with gomplate-like functions
	return processWithFallbackTemplate(templateContent, envData)
}

// gomplateAvailable checks if gomplate binary is available in PATH
func gomplateAvailable() bool {
	_, err := exec.LookPath("gomplate")
	return err == nil
}

// processWithGomplateBinary uses the gomplate command-line tool
func processWithGomplateBinary(templateContent []byte, envData map[string]interface{}) ([]byte, error) {
	cmd := exec.Command("gomplate")
	
	// Set environment variables for gomplate
	env := os.Environ()
	for key, value := range envData {
		env = append(env, fmt.Sprintf("%s=%v", key, value))
	}
	cmd.Env = env
	
	// Pipe template content to gomplate
	cmd.Stdin = bytes.NewReader(templateContent)
	
	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	// Execute gomplate
	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("gomplate execution failed: %w, stderr: %s", err, stderr.String())
	}
	
	return stdout.Bytes(), nil
}

// processWithFallbackTemplate provides basic template processing when gomplate isn't available
func processWithFallbackTemplate(templateContent []byte, envData map[string]interface{}) ([]byte, error) {
	// Create a function map with gomplate-like functions
	funcMap := template.FuncMap{
		"env": func() map[string]interface{} {
			return map[string]interface{}{
				"Getenv": func(key string, defaultValue ...string) string {
					if value := os.Getenv(key); value != "" {
						return value
					}
					if len(defaultValue) > 0 {
						return defaultValue[0]
					}
					return ""
				},
			}
		},
		"getenv": func(key string, defaultValue ...string) string {
			if value := os.Getenv(key); value != "" {
				return value
			}
			if len(defaultValue) > 0 {
				return defaultValue[0]
			}
			return ""
		},
		"toYaml": func(data interface{}) (string, error) {
			yamlData, err := yaml.Marshal(data)
			if err != nil {
				return "", err
			}
			return strings.TrimSuffix(string(yamlData), "\n"), nil
		},
		"nindent": func(indent int, text string) string {
			lines := strings.Split(text, "\n")
			indentStr := strings.Repeat(" ", indent)
			var result []string
			result = append(result, "") // Start with newline
			for _, line := range lines {
				if line != "" {
					result = append(result, indentStr+line)
				} else {
					result = append(result, "")
				}
			}
			return strings.Join(result, "\n")
		},
	}
	
	// Parse template
	tmpl, err := template.New("template").Funcs(funcMap).Parse(string(templateContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}
	
	// Execute template
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, envData)
	if err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}
	
	return buf.Bytes(), nil
}
