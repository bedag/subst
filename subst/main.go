package main

import (
	"runtime/debug"

	"github.com/bedag/subst/subst/cmd"
)

func main() {
	debug.SetGCPercent(100)
	cmd.Execute()
}
