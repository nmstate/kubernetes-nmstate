package appregistry

import (
	"fmt"
	"strings"

	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

// OperatorSourceSpecifier interface provides capability to have different ways
// of specifying operator source via command line flags. This helps us
// maintain backward compatability.
type OperatorSourceSpecifier interface {
	Parse(specifiers []string) ([]*Source, error)
}

type Input struct {
	// Sources is the set of remote operator source(s) specified where operator
	// manifest(s) are located.
	Sources []*Source

	// Packages is the set of package name(s) specified.
	Packages []*Package
}

type Package struct {
	// The name of the package
	Name string
	// The release number of the package
	Release string
}

func (p *Package) String() string {
	if p.Release == "" {
		return fmt.Sprintf("%s", p.Name)
	}
	return fmt.Sprintf("%s:%s", p.Name, p.Release)
}

func (i *Input) IsGoodToProceed() bool {
	return len(i.Sources) > 0 && len(i.Packages) > 0
}

func (i *Input) PackagesToMap() map[Package]bool {
	packages := map[Package]bool{}

	for _, pkg := range i.Packages {
		packages[*pkg] = false
	}

	return packages
}

type inputParser struct {
	sourceSpecifier OperatorSourceSpecifier
}

// Parse parses the raw input provided, sanitizes it and returns an instance of
// Input.
//
// csvSources is a slice of operator source(s) specified. Each operator source
// is expected to be specified as follows.
//
// {base url with cnr prefix}|{quay registry namespace}|{secret namespace/secret name}
//
// csvPackages is a comma separated list of package(s). It is expected to have
// the following format.
// etcd,prometheus,descheduler
//
func (p *inputParser) Parse(csvSources []string, csvPackages string) (*Input, error) {
	sources, err := p.sourceSpecifier.Parse(csvSources)
	if err != nil && len(sources) == 0 {
		return nil, err
	}

	packages, err := sanitizePackageList(strings.Split(csvPackages, ","))

	return &Input{
		Sources:  sources,
		Packages: packages,
	}, err
}

// sanitizePackageList sanitizes the set of package(s) specified. It removes
// duplicates and ignores empty string.
func sanitizePackageList(in []string) ([]*Package, error) {
	out := make([]*Package, 0)
	allErrors := []error{}
	inMap := map[string]bool{}

	for _, item := range in {
		name, release, err := getNameAndRelease(item)
		if err != nil {
			allErrors = append(allErrors, err)
			continue
		}

		if _, ok := inMap[name]; ok || name == "" {
			continue
		}

		inMap[name] = true
		out = append(out, &Package{Name: name, Release: release})
	}

	err := utilerrors.NewAggregate(allErrors)
	return out, err
}

func getNameAndRelease(in string) (string, string, error) {
	inWithoutSpaces := strings.Map(
		func(r rune) rune {
			if r == ' ' {
				return -1
			}
			return r
		},
		in,
	)
	parts := strings.Split(inWithoutSpaces, ":")

	switch len(parts) {
	case 1:
		// release wasn't specified
		return parts[0], "", nil
	case 2:
		return parts[0], parts[1], nil
	default:
		return "", "", fmt.Errorf("Failed to parse package %s", in)
	}
}
