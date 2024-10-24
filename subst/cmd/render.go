package cmd

import (
	"fmt"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/bedag/subst/internal/utils"
	"github.com/bedag/subst/pkg/config"
	"github.com/bedag/subst/pkg/subst"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

func newRenderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "render",
		Short: "Render structure with substitutions",
		Long: heredoc.Doc(`
			Run 'subst discover' to return directories that contain plugin compatible files. Mainly used for automatic plugin discovery by ArgoCD`),
		Example: `# Render the local manifests
subst render 
# Render in a different directory
subst render ../examples/02-overlays/clusters/cluster-01`,
		RunE: render,
	}

	flags := cmd.Flags()
	addCommonFlags(flags)
	addRenderFlags(flags)
	return cmd
}

func addRenderFlags(flags *flag.FlagSet) {
	if flags.Lookup("kubeconfig") == nil {
		flags.String("kubeconfig", "", "Path to a kubeconfig")
	}
	if flags.Lookup("kube-api") == nil {
		flags.String("kube-api", "", "Kubernetes API Url")
	}
	flags.Bool("convert-secret-name", true, heredoc.Doc(`
			Assuming the secret name is derived from ARGOCD_APP_NAME, this option will only use the application name (without project-name_)`))
	flags.Bool("skip-secret-lookup", false, heredoc.Doc(`
		Skip reading from decryption keys from Secret`))
	flags.String("secret-name", "", heredoc.Doc(`
	        Specify Secret name (each key within the secret will be used as a decryption key)`))
	flags.String("secret-namespace", "", heredoc.Doc(`
	        Specify Secret namespace`))
	flags.StringSlice("ejson-key", []string{}, heredoc.Doc(`
			Specify EJSON Private key used for decryption.
			May be specified multiple times or separate values with commas`))
	flags.Bool("skip-decrypt", false, heredoc.Doc(`
			Skip decryption`))
	flags.String("env-regex", "^ARGOCD_ENV_.*$", heredoc.Doc(`
	        Only expose environment variables that match the given regex`))
	flags.String("output", "yaml", heredoc.Doc(`
	        Output format. One of: yaml, json`))

}

func render(cmd *cobra.Command, args []string) error {
	start := time.Now() // Start time measurement

	dir, err := rootDirectory(args)
	if err != nil {
		return err
	}

	configuration, err := config.LoadConfiguration(cfgFile, cmd, dir)
	if err != nil {
		return fmt.Errorf("failed loading configuration: %w", err)
	}
	m, err := subst.New(*configuration)
	if err != nil {
		return err
	}

	err = m.BuildSubstitutions()
	if err != nil {
		return err
	}

	if m != nil {
		err = m.Build()
		if err != nil {
			return err
		}
		if m.Manifests != nil {
			for _, f := range m.Manifests {
				if configuration.Output == "json" {
					err = utils.PrintJSON(f)
					if err != nil {
						log.Error().Msgf("failed to print JSON: %s", err)
					}
				} else {
					err = utils.PrintYAML(f)
					if err != nil {
						log.Error().Msgf("failed to print JSON: %s", err)
					}
				}
			}
		}
	}
	elapsed := time.Since(start) // Calculate elapsed time
	log.Debug().Msgf("Build time for rendering: %s", elapsed)

	return nil
}
