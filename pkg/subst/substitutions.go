package subst

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"

	decrypt "github.com/bedag/subst/internal/decryptors"
	"github.com/bedag/subst/internal/utils"
	"github.com/rs/zerolog/log"
	"sigs.k8s.io/kustomize/api/resmap"
)

var (
	matchingRegex *regexp.Regexp
)

type Substitutions struct {
	Subst      map[interface{}]interface{} `yaml:"subst"`
	Config     SubstitutionsConfig         `yaml:"config"`
	decryptors []decrypt.Decryptor
	Resources  resmap.ResMap
}

type SubstitutionsConfig struct {
	SubstKey         string `yaml:"subst_key"`
	EnvironmentRegex string `yaml:"environment_regex"`
	SubstFileRegex   string `yaml:"subst_file_pattern"`
	FlattenLowerCase bool   `yaml:"lowercase"`
}

func NewSubstitutions(cfg SubstitutionsConfig, decrypts []decrypt.Decryptor, res resmap.ResMap) (s *Substitutions, err error) {

	if cfg.SubstKey == "" {
		cfg.SubstKey = "subst"
	}

	init := &Substitutions{
		Subst:      make(map[interface{}]interface{}),
		Config:     cfg,
		decryptors: decrypts,
		Resources:  res,
	}

	if init.Config.SubstFileRegex != "" {
		matchingRegex, err = regexp.Compile(init.Config.SubstFileRegex)
		if err != nil {
			return nil, err
		}
		log.Debug().Msgf("using regex: %s", init.Config.SubstFileRegex)
	}

	envs, err := GetVariables(cfg.EnvironmentRegex)
	if err != nil {
		return nil, err
	}
	err = init.Add(utils.ToInterface(envs), true)
	if err != nil {
		return nil, err
	}

	return init, nil
}

// Get returns the Substitutions as map[interface{}]interface{}
func (s *Substitutions) Get() map[interface{}]interface{} {
	return s.Subst
}

// ToMap returns the Substitutions as map[string]interface{}
func (s *Substitutions) GetMap() map[string]interface{} {
	return utils.ToMap(s.Subst)
}

// adds new data to the Substitutions
func (s *Substitutions) Add(data map[interface{}]interface{}, optimistic bool) (err error) {

	tree, err := s.Eval(data, nil, optimistic)
	if err != nil {
		return fmt.Errorf("failed to build substitutions: %s", err)
	}

	// Simple deep merge function to replace spruce.Merge
	merged := utils.MergeValues(s.Get(), tree)
	s.Subst = merged
	return nil
}

// Eval evaluates data using gomplate templating
func (s *Substitutions) Eval(data map[interface{}]interface{}, substs map[interface{}]interface{}, optimistic bool) (eval map[interface{}]interface{}, err error) {
	if substs == nil {
		substs = s.Get()
	}

	// Create substitution context
	sub := map[interface{}]interface{}{
		s.Config.SubstKey: substs,
	}

	// Merge data with substitutions
	merged := utils.MergeValues(data, sub)

	// Convert to YAML for processing
	yamlData, err := utils.ToYAML(merged)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to YAML: %w", err)
	}

	// Process with gomplate (using environment regex from config)
	processed, err := utils.TemplateWithEngine([]byte(yamlData), merged, s.Config.EnvironmentRegex)
	if err != nil {
		if optimistic {
			// In optimistic mode, return original data if templating fails
			log.Debug().Msgf("Optimistic mode: templating failed, returning original data: %s", err)
			return merged, nil
		}
		return nil, err
	}

	return processed, nil
}

func (s *Substitutions) Walk(path string, f fs.FileInfo) error {

	if f.IsDir() {
		return nil
	}
	full := filepath.Join(path, f.Name())

	if matchingRegex.MatchString(f.Name()) {
		var c map[interface{}]interface{}
		log.Debug().Msgf("processing: %s", full)
		file, err := utils.NewFile(full)
		if err != nil {
			return err
		}

		// First try to parse as YAML/JSON directly
		c, err = utils.ParseYAML(file.Byte())
		if err != nil {
			// If parsing fails, try template processing
			c, err = utils.Template(file.Byte(), s.Subst)
			if err != nil {
				return fmt.Errorf("failed to process %s: %s", full, err)
			}
		}

		// Read encrypted file
		for _, d := range s.decryptors {
			isEncrypted, _ := d.IsEncrypted(file.Byte())
			if err != nil {
				break
			}
			if isEncrypted {
				log.Debug().Msgf("decrypted: %s", full)
				dm, err := d.Decrypt(file.Byte())
				if err != nil {
					return fmt.Errorf("failed to decrypt %s: %s", full, err)
				}
				t, err := json.Marshal(dm)
				if err != nil {
					return fmt.Errorf("failed to marshal %s: %s", full, err)
				}
				c, err = utils.ParseYAML(t)
				if err != nil {
					return fmt.Errorf("failed to parse %s: %s", full, err)
				}
				break
			}
		}

		if c[resourcesField] != nil {
			log.Debug().Msgf("detected resources in %s", full)
			err = s.addResources(c[resourcesField].([]interface{}))
			if err != nil {
				return fmt.Errorf("failed to add resources from %s: %s", full, err)
			}
			delete(c, resourcesField)
		}

		err = s.Add(c, true)
		if err != nil {
			return fmt.Errorf("failed to merge %s: %s", full, err)
		}

		log.Debug().Msgf("loaded: %s", full)
	}
	return nil
}
