package cmd

import (
	"runtime/debug"
	"testing"
)

func BenchmarkExecute(b *testing.B) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{
		"render",
		"--skip-decrypt",
		"/Users/adrian/git/inventory/clusters/k8s-bedag-root-dev",
	})
	for i := 0; i < b.N; i++ {
		debug.SetGCPercent(100)
		err := cmd.Execute()
		if err != nil {
			b.Errorf("Error: %v", err)
		}
	}
}
