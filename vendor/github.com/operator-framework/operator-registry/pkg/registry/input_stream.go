package registry

import (
	"fmt"
	"sort"

	"github.com/blang/semver"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

type ReplacesInputStream struct {
	graph    GraphLoader
	packages map[string][]*ImageInput
}

func NewReplacesInputStream(graph GraphLoader, toAdd []*ImageInput) (*ReplacesInputStream, error) {
	stream := &ReplacesInputStream{
		graph:    graph,
		packages: map[string][]*ImageInput{},
	}

	// Sort the bundle images into buckets by package
	for _, image := range toAdd {
		pkg := image.Bundle.Package
		stream.packages[pkg] = append(stream.packages[pkg], image)
	}

	// Sort each package bucket by ascending semver
	var errs []error
	for pkg, bundles := range stream.packages {
		var pkgErrs []error
		sort.SliceStable(bundles, func(i, j int) bool {
			less, err := bundleVersionLess(bundles[i].Bundle, bundles[j].Bundle)
			if err != nil {
				pkgErrs = append(pkgErrs, err)
				return false
			}

			return less
		})

		if len(pkgErrs) > 0 {
			// Ignore this package
			delete(stream.packages, pkg)
			errs = append(errs, pkgErrs...)
			continue
		}

		stream.packages[pkg] = bundles
	}

	return stream, utilerrors.NewAggregate(errs)
}

// bundleVersionLess returns true if the version of the first bundle parameter is less than the second, otherwise it returns false.
// An error is returned when either bundle doesn't specify a version or their versions are not valid semver.
func bundleVersionLess(a, b *Bundle) (bool, error) {
	var errs []error
	rawVersionA, err := a.Version()
	if err != nil {
		errs = append(errs, fmt.Errorf("unable to get version for bundle %s: %s", a.Name, err))
	}
	versionA, err := semver.Parse(rawVersionA)
	if err != nil {
		errs = append(errs, fmt.Errorf("unable to parse version for bundle %s: %s", a.Name, err))
	}

	rawVersionB, err := b.Version()
	if err != nil {
		errs = append(errs, fmt.Errorf("unable to get version for bundle %s: %s", b.Name, err))
	}
	versionB, err := semver.Parse(rawVersionB)
	if err != nil {
		errs = append(errs, fmt.Errorf("unable to parse version for bundle %s: %s", b.Name, err))
	}

	if len(errs) > 0 {
		return false, utilerrors.NewAggregate(errs)
	}

	return versionA.LT(versionB), nil
}

// canAdd checks that a new bundle can be added in replaces mode (i.e. the replaces defined for the bundle already exists)
func (r *ReplacesInputStream) canAdd(bundle *Bundle, packageGraph *Package) error {
	replaces, err := bundle.Replaces()
	if err != nil {
		return fmt.Errorf("Invalid bundle replaces: %s", err)
	}

	if replaces != "" && !packageGraph.HasCsv(replaces) {
		// We can't add this until a replacement exists
		return fmt.Errorf("Invalid bundle %s, replaces nonexistent bundle %s", bundle.Name, replaces)
	}

	defaultChannel := bundle.Annotations.DefaultChannelName
	if defaultChannel != "" && !packageGraph.HasChannel(defaultChannel) {
		// We also can't add if the bundle isn't in the default channel it specifies or it doesn't already exist in the package
		var defaultFound bool
		for _, channel := range bundle.Channels {
			if channel != defaultChannel {
				continue
			}
			defaultFound = true
			break
		}

		if !defaultFound {
			return fmt.Errorf("Invalid bundle %s, references nonexistent default channel %s", bundle.Name, defaultChannel)
		}
	}

	images, ok := r.packages[packageGraph.Name]
	if !ok || images == nil {
		// This shouldn't happen unless canAdd is being called without the correct setup
		panic(fmt.Sprintf("Programmer error: package graph %s incorrectly initialized", packageGraph.Name))
	}

	// No edges to any remaining input bundles, this bundle can be added
	return nil
}

// Next returns the next available bundle image from the stream, returning a nil image if the stream is exhausted.
func (r *ReplacesInputStream) Next() (*ImageInput, error) {
	var errs []error
	for pkg, images := range r.packages {
		if len(images) < 1 {
			// No more images to add for this package, clean up
			delete(r.packages, pkg)
			continue
		}

		packageGraph, err := r.graph.Generate(pkg)
		if err != nil {
			if err != ErrPackageNotInDatabase {
				// Can't parse this package any further
				delete(r.packages, pkg)
				errs = append(errs, err)
				continue
			}

			// Adding a brand new package is a different story
			packageGraph = &Package{Name: pkg}
		}

		// Find the next bundle in topological order
		var packageErrs []error
		for i, image := range images {
			if err := r.canAdd(image.Bundle, packageGraph); err != nil {
				// Can't parse this bundle any further right now
				packageErrs = append(packageErrs, err)
				continue
			}

			// Found something we can add
			r.packages[pkg] = append(r.packages[pkg][:i], r.packages[pkg][i+1:]...)
			if len(r.packages[pkg]) < 1 {
				// Remove package if exhausted
				delete(r.packages, pkg)
			}

			return image, nil
		}

		// No viable bundle found in the package, can't parse it any further, so return any errors
		errs = append(errs, packageErrs...)
	}

	// We've exhausted all valid input bundles, any errors here indicate invalid input of some kind
	return nil, utilerrors.NewAggregate(errs)
}

// Empty returns true if there are no bundles in the stream.
func (r *ReplacesInputStream) Empty() bool {
	return len(r.packages) < 1
}
