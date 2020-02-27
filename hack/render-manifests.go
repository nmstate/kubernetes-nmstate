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

	tmpl, err := template.ParseGlob(inputDir + "/*.yaml")
	if err != nil {
		exitWithError(err, "failed parsing manifests at %s", inputDir)
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
