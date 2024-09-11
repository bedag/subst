package subst

import (
	"context"
	"fmt"
	"sync"

	decrypt "github.com/bedag/subst/internal/decryptors"
	ejson "github.com/bedag/subst/internal/decryptors/ejson"
	"github.com/bedag/subst/internal/kustomize"
	"github.com/bedag/subst/internal/utils"
	"github.com/bedag/subst/pkg/config"
	"github.com/rs/zerolog/log"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

type Build struct {
	Manifests     []map[interface{}]interface{}
	Kustomization *kustomize.Kustomize
	Substitutions *Substitutions
	cfg           config.Configuration
	kubeClient    *kubernetes.Clientset
}

func New(config config.Configuration) (build *Build, err error) {

	k, err := kustomize.NewKustomize(config.RootDirectory)
	if err != nil {
		return nil, err
	}

	init := &Build{
		cfg:           config,
		Kustomization: k,
	}

	return init, err
}

func (b *Build) BuildSubstitutions() (err error) {
	decryptors, cleanups, err := b.decryptors()
	if err != nil {
		return err
	}

	defer func() {
		for _, cleanup := range cleanups {
			cleanup()
		}
	}()

	SubstitutionsConfig := SubstitutionsConfig{
		EnvironmentRegex: b.cfg.EnvRegex,
		SubstFileRegex:   b.cfg.FileRegex,
	}

	b.Substitutions, err = NewSubstitutions(SubstitutionsConfig, decryptors, b.Kustomization.Build)
	if err != nil {
		return err
	}

	err = b.loadSubstitutions()
	if err != nil {
		return err
	}
	return nil

}

func (b *Build) Build() (err error) {

	if b.Substitutions == nil {
		log.Debug().Msg("no resources to build")
		return nil
	}

	decryptors, cleanups, err := b.decryptors()
	if err != nil {
		return err
	}

	defer func() {
		for _, cleanup := range cleanups {
			cleanup()
		}
	}()

	// Run Build
	log.Debug().Msg("substitute manifests")

	var wg sync.WaitGroup
	manifestsMutex := sync.Mutex{}
	for _, manifest := range b.Substitutions.Resources.Resources() {
		wg.Add(1)
		go func() {
			defer wg.Done()

			var c map[interface{}]interface{}

			log.Debug().Msg("Before marshalling")
			mBytes, _ := manifest.MarshalJSON()
			// should not check every file if its encrypted
			// already decrypted in substiqutions.go?
			log.Debug().Msg("Before decryption")
			for _, d := range decryptors {
				isEncrypted, err := d.IsEncrypted(mBytes)
				if err != nil {
					log.Error().Msgf("Error checking encryption for %s: %s", mBytes, err)
					continue
				}
				if isEncrypted {
					dm, err := d.Decrypt(mBytes)
					if err != nil {
						log.Error().Msgf("failed to decrypt %s: %s", mBytes, err)
						return
					}
					c = utils.ToInterface(dm)
					break
				}
			}
			log.Debug().Msg("After decryption")

			if c == nil {
				log.Debug().Msg("AsYAML")
				m, _ := manifest.AsYAML()

				log.Debug().Msg("ParseYAML")
				c, err = utils.ParseYAML(m)
				if err != nil {
					log.Error().Msgf("UnmarshalJSON: %s", err)
					return
				}
			}

			log.Debug().Msg("Before Substitutions.Eval")
			f, err := b.Substitutions.Eval(c, nil, false)
			if err != nil {
				log.Error().Msgf("spruce evaluation failed %s/%s: %s", manifest.GetNamespace(), manifest.GetName(), err)
				return
			}
			log.Debug().Msg("Append to Manifest")
			manifestsMutex.Lock()
			b.Manifests = append(b.Manifests, f)
			manifestsMutex.Unlock()
		}()
	}

	wg.Wait()

	return nil
}

// builds the substitutions interface
func (b *Build) loadSubstitutions() (err error) {

	// Read Substition Files
	err = b.Kustomization.Walk(b.Substitutions.Walk)
	if err != nil {
		return err
	}

	// Final attempt to evaluate
	eval, err := b.Substitutions.Eval(b.Substitutions.Subst, nil, false)
	if err != nil {
		return fmt.Errorf("spruce evaluation failed: %s", err)
	}
	b.Substitutions.Subst = eval

	if len(b.Substitutions.Subst) > 0 {
		log.Debug().Msgf("loaded substitutions: %+v", b.Substitutions.Subst)
	} else {
		log.Debug().Msg("no substitutions found")
	}

	return nil
}

// initialize decryption
func (b *Build) decryptors() (decryptors []decrypt.Decryptor, cleanups []func(), err error) {

	c := decrypt.DecryptorConfig{
		SkipDecrypt: b.cfg.SkipDecrypt,
	}

	ed, err := ejson.NewEJSONDecryptor(c, "", b.cfg.EjsonKey...)
	if err != nil {
		return nil, nil, err
	}
	decryptors = append(decryptors, ed)

	if b.cfg.SecretSkip {
		return
	}

	if !b.cfg.SkipDecrypt && (b.cfg.SecretName != "" && b.cfg.SecretNamespace != "") {

		var host string
		if b.cfg.KubeAPI != "" {
			host = b.cfg.KubeAPI
		}
		cfg, err := clientcmd.BuildConfigFromFlags(host, b.cfg.Kubeconfig)
		if err == nil {
			b.kubeClient, err = kubernetes.NewForConfig(cfg)
			if err != nil {
				log.Debug().Msgf("could not load kubernetes client: %s", err)
			} else {
				ctx := context.Background()
				for _, decr := range decryptors {
					err = decr.KeysFromSecret(b.cfg.SecretName, b.cfg.SecretNamespace, b.kubeClient, ctx)
					if err != nil {
						log.Debug().Msgf("failed to load secrets from Kubernetes: %s", err)
					}
				}

			}
		}
	}

	return
}
