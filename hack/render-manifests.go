package main

import (
	"fmt"
	"os"
	"path"
	"text/template"

	"github.com/pkg/errors"
)

func exitWithError(err error, cause string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "render-manifests.go: error: %v\n", errors.Wrapf(err, cause, args))
	os.Exit(1)
}

func main() {
	type Inventory struct {
		Namespace           string
		NMStateHandlerImage string
		ImagePullPolicy     string
	}

	inventory := Inventory{
		Namespace:           os.Args[1],
		NMStateHandlerImage: os.Args[2],
		ImagePullPolicy:     os.Args[3],
	}
	inputDir := os.Args[4]
	outputDir := os.Args[5]

	// Clean up output dir so we don't have old files.
	err := os.RemoveAll(outputDir)
	if err != nil {
		exitWithError(err, "failed cleaning up output dir %s", outputDir)
	}

	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		exitWithError(err, "failed to create output dir %s", outputDir)
	}

	tmpl, err := template.ParseGlob(path.Join(inputDir, "*.yaml"))
	if err != nil {
		exitWithError(err, "failed parsing top dir manifests at %s", inputDir)
	}

	tmpl, err = tmpl.ParseGlob(path.Join(inputDir, "*/*.yaml"))
	if err != nil {
		exitWithError(err, "failed parsing sub dir manifests at %s", inputDir)
	}

	for _, t := range tmpl.Templates() {
		outputFile := path.Join(outputDir, t.Name())
		f, err := os.Create(outputFile)
		if err != nil {
			exitWithError(err, "failed creating expanded template %s", outputFile)
		}

		err = t.Execute(f, inventory)
		if err != nil {
			exitWithError(err, "failed expanding template %s", tmpl)
		}
	}
}
