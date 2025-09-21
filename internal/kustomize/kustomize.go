package kustomize

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Kustomize struct {
	Root    string
	Paths   []string
	BuildYAML string
}

func NewKustomize(root string) (*Kustomize, error) {
	k := &Kustomize{Root: root}
	if err := k.build(); err != nil {
		return nil, err
	}
	if err := k.findKustomizationFiles(); err != nil {
		return nil, err
	}
	return k, nil
}

func (k *Kustomize) build() error {
	// Use kustomize binary instead of library
	cmd := exec.Command("kustomize", "build", k.Root)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("kustomize build failed: %w, stderr: %s", err, stderr.String())
	}
	
	k.BuildYAML = stdout.String()
	return nil
}

func (k *Kustomize) findKustomizationFiles() error {
	// Find all kustomization files in the directory tree
	err := filepath.WalkDir(k.Root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		if d.IsDir() {
			return nil
		}
		
		filename := strings.ToLower(d.Name())
		if filename == "kustomization.yaml" || filename == "kustomization.yml" || filename == "kustomization" {
			k.Paths = append(k.Paths, filepath.Dir(path))
		}
		
		return nil
	})
	
	return err
}

func (k *Kustomize) GetYAML() string {
	return k.BuildYAML
}

func (k *Kustomize) GetPaths() []string {
	return k.Paths
}
