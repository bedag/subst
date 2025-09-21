package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/MakeNowJust/heredoc"
	"github.com/bedag/subst/internal/wrapper"
	"github.com/bedag/subst/pkg/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

func newDiscoverCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "discover",
		Short: "Discover if plugin is applicable to the given directory",
		Long: heredoc.Doc(`
			Run 'subst discover' to return directories that contain plugin compatible files. Mainly used for automatic plugin discovery by ArgoCD`),
		RunE: discover,
	}

	flags := cmd.Flags()
	addCommonFlags(flags)
	return cmd

}

func discover(cmd *cobra.Command, args []string) error {
	dir, err := rootDirectory(args)
	if err != nil {
		return err
	}

	configuration, err := config.LoadConfiguration(cfgFile, cmd, dir)
	if err != nil {
		return fmt.Errorf("failed loading configuration: %w", err)
	}

	// Check for kustomization.yaml (primary compatibility)
	if hasKustomization(dir) {
		log.Debug().Msg("Found kustomization.yaml - subst plugin applicable")
		fmt.Println("subst")
		return nil
	}

	// Check for template files (including gomplate templates)
	if hasTemplateFiles(dir, configuration.FileRegex) {
		log.Debug().Msg("Found template files - subst plugin applicable")
		fmt.Println("subst")
		return nil
	}

	// Check for any YAML files that might contain templates
	if hasYAMLWithTemplates(dir) {
		log.Debug().Msg("Found YAML files with template syntax - subst plugin applicable")
		fmt.Println("subst")
		return nil
	}

	log.Debug().Msg("No applicable files found")
	return fmt.Errorf("no applicable files found in directory %s", dir)
}

// hasKustomization checks if directory contains kustomization.yaml
func hasKustomization(dir string) bool {
	kustomizationPaths := []string{
		filepath.Join(dir, "kustomization.yaml"),
		filepath.Join(dir, "kustomization.yml"),
		filepath.Join(dir, "Kustomization"),
	}

	for _, path := range kustomizationPaths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

// hasGomplateTemplates checks if directory contains gomplate template files
func hasYAMLWithTemplates(dir string) bool {
	found := false
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check YAML files for template syntax
		if isYAMLFile(info.Name()) {
			content, err := os.ReadFile(path)
			if err != nil {
				return nil // Continue on read error
			}

			if wrapper.IsGomplateTemplate(content) {
				log.Debug().Msgf("Found YAML with template syntax: %s", path)
				found = true
				return filepath.SkipDir // Stop walking once found
			}
		}

		return nil
	})

	if err != nil {
		log.Error().Msgf("Error walking directory: %s", err)
	}

	return found
}

// hasGomplateTemplates checks if directory contains gomplate template files (keeping for compatibility)
func hasGomplateTemplates(dir string, gomplateRegex string) bool {
	if gomplateRegex == "" {
		gomplateRegex = `\.(gotmpl|tmpl)$`
	}

	regex, err := regexp.Compile(gomplateRegex)
	if err != nil {
		log.Error().Msgf("Invalid gomplate regex pattern: %s", err)
		return false
	}

	found := false
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check by file extension first
		if regex.MatchString(info.Name()) {
			log.Debug().Msgf("Found gomplate template by extension: %s", path)
			found = true
			return filepath.SkipDir // Stop walking once found
		}

		// Check by content analysis for files that might be gomplate templates
		if isLikelyTemplateFile(info.Name()) {
			content, err := os.ReadFile(path)
			if err != nil {
				return nil // Continue on read error
			}

			if wrapper.IsGomplateTemplate(content) {
				log.Debug().Msgf("Found gomplate template by content: %s", path)
				found = true
				return filepath.SkipDir // Stop walking once found
			}
		}

		return nil
	})

	if err != nil {
		log.Error().Msgf("Error walking directory: %s", err)
	}

	return found
}

// hasTemplateFiles checks if directory contains template files matching the pattern
func hasTemplateFiles(dir string, fileRegex string) bool {
	if fileRegex == "" {
		return false
	}

	regex, err := regexp.Compile(fileRegex)
	if err != nil {
		log.Error().Msgf("Invalid file regex pattern: %s", err)
		return false
	}

	found := false
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if regex.MatchString(info.Name()) {
			log.Debug().Msgf("Found template file: %s", path)
			found = true
			return filepath.SkipDir // Stop walking once found
		}

		return nil
	})

	if err != nil {
		log.Error().Msgf("Error walking directory: %s", err)
	}

	return found
}

// isLikelyTemplateFile checks if a file might be a template based on its name
func isLikelyTemplateFile(filename string) bool {
	templateExtensions := []string{
		".yaml", ".yml", ".json", ".toml", ".txt", ".conf", ".config",
	}

	for _, ext := range templateExtensions {
		if filepath.Ext(filename) == ext {
			return true
		}
	}
	return false
}

// isYAMLFile checks if a file is a YAML file
func isYAMLFile(filename string) bool {
	ext := filepath.Ext(filename)
	return ext == ".yaml" || ext == ".yml"
}
