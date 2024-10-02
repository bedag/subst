package cmd

import (
	"fmt"

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
		if len(m.Substitutions.Subst) > 0 {
			if configuration.Output == "json" {
				err = utils.PrintJSON(m.Substitutions.Subst)
				if err != nil {
					log.Error().Msgf("failed to print JSON: %s", err)
				}
			} else {
				err = utils.PrintYAML(m.Substitutions.Subst)
				if err != nil {
					log.Error().Msgf("failed to print JSON: %s", err)
				}
			}
		}
	}

	return nil
}
