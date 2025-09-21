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
)

func newSubstitutionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "substitutions",
		Short: "Render available substitutions",
		Long: heredoc.Doc(`
			Run 'subst substitutions' to return available substitutions for given Kustomize.`),
		RunE: substitutions,
	}

	flags := cmd.Flags()
	addCommonFlags(flags)
	addRenderFlags(flags)
	return cmd

}

func substitutions(cmd *cobra.Command, args []string) error {
	start := time.Now() // Start time measurement

	dir, err := rootDirectory(args)
	if err != nil {
		return err
	}

	configuration, err := config.LoadConfiguration(cfgFile, cmd, dir)
	if err != nil {
		return fmt.Errorf("failed loading configuration: %w", err)
	}
	m, err := subst.NewSubst(*configuration)
	if err != nil {
		return err
	}

	err = m.BuildSubstitutions()
	if err != nil {
		return err
	}

	if m != nil {
		if len(m.Substitutions) > 0 {
			// Convert to interface{} map for compatibility with utils functions
			substMap := make(map[interface{}]interface{})
			for k, v := range m.Substitutions {
				substMap[k] = v
			}
			
			if configuration.Output == "json" {
				err = utils.PrintJSON(substMap)
				if err != nil {
					log.Error().Msgf("failed to print JSON: %s", err)
				}
			} else {
				err = utils.PrintYAML(substMap)
				if err != nil {
					log.Error().Msgf("failed to print YAML: %s", err)
				}
			}
		}
	}
	elapsed := time.Since(start) // Calculate elapsed time
	log.Debug().Msgf("Build time for substitutions: %s", elapsed)

	return nil
}
