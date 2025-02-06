/*
Copyright The Kubernetes NMState Authors.


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"text/template"

	"github.com/pkg/errors"
)

func exitWithError(err error, cause string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "render-manifests.go: error: %v\n", errors.Wrapf(err, cause, args...))
	os.Exit(1)
}

func main() {
	type Inventory struct {
		HandlerNamespace    string
		HandlerImage        string
		HandlerPullPolicy   string
		HandlerPrefix       string
		OperatorNamespace   string
		OperatorImage       string
		OperatorPullPolicy  string
		MonitoringNamespace string
		KubeRBACProxyImage  string
	}

	handlerNamespace := flag.String("handler-namespace", "nmstate", "Namespace for the NMState handler")
	handlerImage := flag.String("handler-image", "", "Image for the NMState handler")
	handlerPullPolicy := flag.String("handler-pull-policy", "Always", "Pull policy for the NMState handler image")
	handlerPrefix := flag.String("handler-prefix", "", "Name prefix for the NMState handler's resources")
	operatorNamespace := flag.String("operator-namespace", "nmstate-operator", "Namespace for the NMState operator")
	operatorImage := flag.String("operator-image", "", "Image for the NMState operator")
	operatorPullPolicy := flag.String("operator-pull-policy", "Always", "Pull policy for the NMState operator image")
	monitoringNamespace := flag.String("monitoring-namespace", "monitoring", "Namespace for the cluster monitoring")
	kubeRBACProxyImage := flag.String("kube-rbac-proxy-image", "", "Image for the kube RBAC proxy needed for metrics")
	inputDir := flag.String("input-dir", "", "Input directory")
	outputDir := flag.String("output-dir", "", "Output directory")
	flag.Parse()

	inventory := Inventory{
		HandlerNamespace:    *handlerNamespace,
		HandlerImage:        *handlerImage,
		HandlerPullPolicy:   *handlerPullPolicy,
		HandlerPrefix:       *handlerPrefix,
		OperatorNamespace:   *operatorNamespace,
		OperatorImage:       *operatorImage,
		OperatorPullPolicy:  *operatorPullPolicy,
		MonitoringNamespace: *monitoringNamespace,
		KubeRBACProxyImage:  *kubeRBACProxyImage,
	}

	// Clean up output dir so we don't have old files.
	err := os.RemoveAll(*outputDir)
	if err != nil {
		exitWithError(err, "failed cleaning up output dir %s", *outputDir)
	}

	err = os.MkdirAll(*outputDir, 0755) //nolint:mnd
	if err != nil {
		exitWithError(err, "failed to create output dir %s", *outputDir)
	}

	// Be explicit about which subdirs we render. Otherwise, we might inadvertently override
	// a manifest with the same name.
	var tmpl *template.Template
	tmpl, err = template.ParseGlob(path.Join(*inputDir, "operator/*.yaml"))
	if err != nil {
		exitWithError(err, "failed parsing top dir operator manifests at %s", *inputDir)
	}

	tmpl, err = tmpl.ParseGlob(path.Join(*inputDir, "examples/*.yaml"))
	if err != nil {
		exitWithError(err, "failed parsing sub dir example manifests at %s", *inputDir)
	}

	for _, t := range tmpl.Templates() {
		outputFile := path.Join(*outputDir, t.Name())
		f, err := os.Create(outputFile)
		if err != nil {
			exitWithError(err, "failed creating expanded template %s", outputFile)
		}

		err = t.Execute(f, inventory)
		if err != nil {
			exitWithError(err, "failed expanding template %+v", tmpl)
		}
	}
}
