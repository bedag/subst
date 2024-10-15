package main

import (
	"log/slog"

	"github.com/KimMachineGun/automemlimit/memlimit"
	"github.com/bedag/subst/subst/cmd"

	_ "go.uber.org/automaxprocs"
)

func main() {
	memlimit.SetGoMemLimitWithOpts(
		memlimit.WithRatio(0.9),
		memlimit.WithProvider(
			memlimit.ApplyFallback(
				memlimit.FromCgroup,
				memlimit.FromSystem,
			),
		),
		memlimit.WithLogger(slog.Default()),
	)
	cmd.Execute()
}
