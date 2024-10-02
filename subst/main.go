package main

import (
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
	)
	cmd.Execute()
}
