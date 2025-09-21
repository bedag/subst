package subst

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bedag/subst/internal/kustomize"
	"github.com/bedag/subst/internal/wrapper"
	"github.com/bedag/subst/pkg/config"
	"github.com/rs/zerolog/log"
	"gopkg.in/yaml.v3"
)

// Subst represents a simplified subst processor for CMP
type Subst struct {
	Kustomization *kustomize.Kustomize
	Manifests     [][]byte // Store as byte slices for simplicity
	Substitutions map[string]interface{}
}

// NewSubst creates a new simplified Subst instance
func NewSubst(config config.Configuration) (*Subst, error) {
	k, err := kustomize.NewKustomize(config.RootDirectory)
	if err != nil {
		return nil, err
	}

	// Get environment variables that match the regex
	envVars, err := GetVariables(config.EnvRegex)
	if err != nil {
		return nil, err
	}

	subst := &Subst{
		Kustomization: k,
		Manifests:     [][]byte{},
		Substitutions: envVars,
	}

	// Load .vars files from kustomize paths
	err = subst.loadVarsFiles()
	if err != nil {
		log.Warn().Msgf("Failed to load subst files: %v", err)
	}

	return subst, nil
}

// loadVarsFiles loads subst.yaml files from the kustomize directory tree and parent directories
func (s *Subst) loadVarsFiles() error {
	paths := s.Kustomization.GetPaths()
	
	// For each kustomize path, also check parent directories for subst files
	allPaths := make(map[string]bool)
	for _, path := range paths {
		allPaths[path] = true
		
		// Also check parent directories for common subst files
		currentPath := path
		for i := 0; i < 5; i++ { // Limit depth to avoid infinite loops
			parentPath := filepath.Dir(currentPath)
			if parentPath == currentPath || parentPath == "/" {
				break
			}
			allPaths[parentPath] = true
			currentPath = parentPath
		}
	}
	
	// Load subst files from all paths
	for path := range allPaths {
		err := s.loadSubstFromPath(path)
		if err != nil {
			log.Warn().Msgf("Failed to load subst files from %s: %v", path, err)
		}
	}
	
	return nil
}

// loadSubstFromPath loads subst.yaml files from a specific path
func (s *Subst) loadSubstFromPath(basePath string) error {
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return err
	}
	
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		if entry.Name() == "subst.yaml" {
			filePath := filepath.Join(basePath, entry.Name())
			log.Debug().Msgf("Loading subst file: %s", filePath)
			substData, err := s.loadSubstFile(filePath)
			if err != nil {
				log.Warn().Msgf("Failed to load %s: %v", filePath, err)
				continue
			}
			
			// Merge subst data into substitutions
			for k, v := range substData {
				s.Substitutions[k] = v
			}
		}
	}
	
	return nil
}

// loadSubstFile loads a single subst.yaml file and extracts the settings section
func (s *Subst) loadSubstFile(filePath string) (map[string]interface{}, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	var substFile map[string]interface{}
	err = yaml.Unmarshal(content, &substFile)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", filePath, err)
	}
	
	// Extract the settings section - this contains the substitution data
	if settings, ok := substFile["settings"].(map[string]interface{}); ok {
		return settings, nil
	}
	
	// If no settings section, return empty map (not an error)
	log.Debug().Msgf("No settings section found in %s", filePath)
	return make(map[string]interface{}), nil
}

// loadVarsFile loads a single .vars file (legacy support)
func (s *Subst) loadVarsFile(filePath string) (map[string]interface{}, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	
	var vars map[string]interface{}
	err = yaml.Unmarshal(content, &vars)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", filePath, err)
	}
	
	return vars, nil
}

// BuildSubstitutions loads substitution variables
func (s *Subst) BuildSubstitutions() error {
	// For maximum simplicity, just use environment variables
	// In future, this could be extended to load from .vars files
	log.Debug().Msgf("Using %d environment substitutions", len(s.Substitutions))
	return nil
}

// Build processes kustomize output with gomplate templates
func (s *Subst) Build() error {
	if s.Kustomization == nil {
		return fmt.Errorf("no kustomization configured")
	}

	log.Debug().Msg("Building resources with simplified approach")

	// Get YAML output from kustomize
	yamlContent := s.Kustomization.GetYAML()
	if yamlContent == "" {
		return fmt.Errorf("kustomize produced no output")
	}

	// Structure the data for gomplate processing matching Spruce structure
	templateData := make(map[string]interface{})
	vars := make(map[string]interface{})
	secretsData := make(map[string]interface{})
	settings := make(map[string]interface{})
	
	for k, v := range s.Substitutions {
		log.Debug().Msgf("Processing substitution: %s = %+v", k, v)
		
		if k == "secrets" {
			// Handle secrets section from subst.yaml - put under settings
			if secretsMap, ok := v.(map[string]interface{}); ok {
				if data, ok := secretsMap["data"].(map[string]interface{}); ok {
					log.Debug().Msg("Adding secrets.data from subst.yaml")
					for secretKey, secretValue := range data {
						secretsData[secretKey] = secretValue
					}
				}
			}
		} else if k == "vars" {
			// Handle vars section from subst.yaml  
			if varsMap, ok := v.(map[string]interface{}); ok {
				log.Debug().Msg("Adding vars from subst.yaml")
				for varKey, varValue := range varsMap {
					vars[varKey] = varValue
				}
			}
		} else if strings.HasPrefix(k, "ARGOCD_ENV_") {
			// Handle environment variables
			cleanKey := strings.TrimPrefix(k, "ARGOCD_ENV_")
			
			if strings.Contains(cleanKey, "password") || strings.Contains(cleanKey, "secret") {
				log.Debug().Msgf("Adding %s to secrets.data", cleanKey)
				secretsData[cleanKey] = v
			} else if strings.HasPrefix(cleanKey, "cluster_") {
				clusterKey := strings.TrimPrefix(cleanKey, "cluster_")
				if cluster, ok := vars["cluster"].(map[string]interface{}); ok {
					cluster[clusterKey] = v
				} else {
					vars["cluster"] = map[string]interface{}{clusterKey: v}
				}
			} else {
				log.Debug().Msgf("Adding %s to vars", cleanKey)
				vars[cleanKey] = v
			}
		} else if strings.Contains(k, "password") || strings.Contains(k, "secret") {
			// Put secrets-like data into secrets.data
			log.Debug().Msgf("Adding %s to secrets.data", k)
			secretsData[k] = v
		} else if strings.HasPrefix(k, "cluster_") {
			// Put cluster vars into cluster object
			clusterKey := strings.TrimPrefix(k, "cluster_")
			log.Debug().Msgf("Adding %s to vars.cluster.%s", k, clusterKey)
			if cluster, ok := vars["cluster"].(map[string]interface{}); ok {
				cluster[clusterKey] = v
			} else {
				vars["cluster"] = map[string]interface{}{clusterKey: v}
			}
		} else {
			// Other vars go directly into vars
			log.Debug().Msgf("Adding %s to vars", k)
			vars[k] = v
		}
	}
	
	// Set up the settings structure (like subst.yaml structure)
	if len(vars) > 0 {
		settings["vars"] = vars
	}
	if len(secretsData) > 0 {
		settings["secrets"] = map[string]interface{}{
			"data": secretsData,
		}
	}
	
	// Set up the template data structure with settings
	templateData["vars"] = vars // Keep vars at top level for $.subst.vars.* syntax
	if len(secretsData) > 0 {
		templateData["secrets"] = map[string]interface{}{
			"data": secretsData,
		}
	}
	if len(settings) > 0 {
		templateData["settings"] = settings // Add settings for $.subst.settings.* syntax
	}

	log.Debug().Msgf("Template data: %+v", templateData)

	// Process the entire YAML content with gomplate
	processedYAMLBytes, err := wrapper.ProcessGomplateTemplate([]byte(yamlContent), templateData, "")
	if err != nil {
		return fmt.Errorf("failed to process with gomplate: %w", err)
	}

	// For maximum simplicity, treat the entire output as one document
	// This works fine for ArgoCD CMP which just needs valid YAML output
	s.Manifests = [][]byte{processedYAMLBytes}

	log.Debug().Msgf("Built %d manifest(s)", len(s.Manifests))
	return nil
}
