package declcfg

import (
	"fmt"
	"reflect"
	"sort"
	"sync"

	"github.com/blang/semver/v4"
	"github.com/mitchellh/hashstructure/v2"
	"github.com/sirupsen/logrus"

	"github.com/operator-framework/operator-registry/alpha/model"
	"github.com/operator-framework/operator-registry/alpha/property"
)

// DiffGenerator configures how diffs are created via Run().
type DiffGenerator struct {
	Logger *logrus.Entry

	// SkipDependencies directs Run() to not include dependencies
	// of bundles included in the diff if true.
	SkipDependencies bool
	// Includer for adding catalog objects to Run() output.
	Includer DiffIncluder
	// IncludeAdditively catalog objects specified in Includer in headsOnly mode.
	IncludeAdditively bool

	initOnce sync.Once
}

func (g *DiffGenerator) init() {
	g.initOnce.Do(func() {
		if g.Logger == nil {
			g.Logger = &logrus.Entry{}
		}
		if g.Includer.Logger == nil {
			g.Includer.Logger = g.Logger
		}
	})
}

// Run returns a Model containing a subset of catalog objects in newModel:
// - If g.Includer contains objects:
//   - If g.IncludeAdditively is false, a diff will be generated only on those objects,
//     depending on the mode.
//   - If g.IncludeAdditionally is true, the diff will contain included objects,
//     plus those added by the mode.
// - If in heads-only mode (oldModel == nil), then the heads of channels are added to the output.
// - If in latest mode, a diff between old and new Models is added to the output.
// - Dependencies are added in all modes if g.SkipDependencies is false.
func (g *DiffGenerator) Run(oldModel, newModel model.Model) (model.Model, error) {
	g.init()

	// TODO(estroz): loading both oldModel and newModel into memory may
	// exceed process/hardware limits. Instead, store models on-disk then
	// load by package.

	outputModel := model.Model{}

	// Prunes old objects from outputModel if they exist.
	latestPruneFromOutput := func() error {

		for _, outputPkg := range outputModel {
			oldPkg, oldHasPkg := oldModel[outputPkg.Name]
			if !oldHasPkg {
				// outputPkg was already copied to outputModel above.
				continue
			}
			if err := pruneOldFromNewPackage(oldPkg, outputPkg); err != nil {
				return err
			}
			if len(outputPkg.Channels) == 0 {
				// Remove empty packages.
				delete(outputModel, outputPkg.Name)
			}
		}

		return nil
	}

	headsOnlyMode := len(oldModel) == 0
	latestMode := !headsOnlyMode
	isInclude := len(g.Includer.Packages) != 0

	switch {
	case !g.IncludeAdditively && isInclude: // Only diff between included objects.

		// Add included packages/channels/bundles from newModel to outputModel.
		if err := g.Includer.Run(newModel, outputModel); err != nil {
			return nil, err
		}

		if latestMode {
			if err := latestPruneFromOutput(); err != nil {
				return nil, err
			}
		}

	case isInclude: // Add included objects to outputModel.

		// Add included packages/channels/bundles from newModel to outputModel.
		if err := g.Includer.Run(newModel, outputModel); err != nil {
			return nil, err
		}

		fallthrough
	default:

		if headsOnlyMode { // Net-new diff of heads only.

			// Make shallow copies of packages and channels that are only
			// filled with channel heads.
			for _, newPkg := range newModel {
				// This package may have been created in the include step.
				outputPkg, pkgIncluded := outputModel[newPkg.Name]
				if !pkgIncluded {
					outputPkg = copyPackageNoChannels(newPkg)
					outputModel[outputPkg.Name] = outputPkg
				}
				for _, newCh := range newPkg.Channels {
					if _, chIncluded := outputPkg.Channels[newCh.Name]; chIncluded {
						// Head (and other bundles) were added in the include step.
						continue
					}
					outputCh := copyChannelNoBundles(newCh, outputPkg)
					outputPkg.Channels[outputCh.Name] = outputCh
					head, err := newCh.Head()
					if err != nil {
						return nil, err
					}
					outputBundle := copyBundle(head, outputCh, outputPkg)
					outputModel.AddBundle(*outputBundle)
				}
			}

		} else { // Diff between old and new Model.

			// Copy newModel to create an output model by deletion,
			// which is more succinct than by addition.
			for _, newPkg := range newModel {
				if _, pkgIncluded := outputModel[newPkg.Name]; pkgIncluded {
					// The user has specified the state they want this package to have in the diff
					// via an inclusion entry, so the package created above should not be changed.
					continue
				}
				outputModel[newPkg.Name] = copyPackage(newPkg)
			}

			if err := latestPruneFromOutput(); err != nil {
				return nil, err
			}

		}

	}

	if !g.SkipDependencies {
		// Add dependencies to outputModel not already present in oldModel.
		if err := addAllDependencies(newModel, oldModel, outputModel); err != nil {
			return nil, err
		}
	}

	// Default channel may not have been copied, so set it to the new default channel here.
	for _, outputPkg := range outputModel {
		newPkg := newModel[outputPkg.Name]
		var outputHasDefault bool
		outputPkg.DefaultChannel, outputHasDefault = outputPkg.Channels[newPkg.DefaultChannel.Name]
		if !outputHasDefault {
			// Create a name-only channel since oldModel contains the channel already.
			outputPkg.DefaultChannel = copyChannelNoBundles(newPkg.DefaultChannel, outputPkg)
		}
	}

	return outputModel, nil
}

// pruneOldFromNewPackage prune any bundles and channels from newPkg that
// are in oldPkg, but not those that differ in any way.
func pruneOldFromNewPackage(oldPkg, newPkg *model.Package) error {
	for _, newCh := range newPkg.Channels {
		oldCh, oldHasCh := oldPkg.Channels[newCh.Name]
		if !oldHasCh {
			// newCh is assumed to have been copied to outputModel by the caller.
			continue
		}

		for _, newBundle := range newCh.Bundles {
			oldBundle, oldHasBundle := oldCh.Bundles[newBundle.Name]
			if !oldHasBundle {
				// newBundle is copied to outputModel by the caller if it is a channel head.
				continue
			}
			equal, err := bundlesEqual(oldBundle, newBundle)
			if err != nil {
				return err
			}
			if equal {
				delete(newCh.Bundles, newBundle.Name)
			}
		}
		if len(newCh.Bundles) == 0 {
			// Remove empty channels.
			delete(newPkg.Channels, newCh.Name)
		}
	}

	return nil
}

// bundlesEqual computes then compares the hashes of b1 and b2 for equality.
func bundlesEqual(b1, b2 *model.Bundle) (bool, error) {
	// Use a declarative config bundle type to avoid infinite recursion.
	dcBundle1 := convertFromModelBundle(b1)
	dcBundle2 := convertFromModelBundle(b2)

	hash1, err := hashstructure.Hash(dcBundle1, hashstructure.FormatV2, nil)
	if err != nil {
		return false, err
	}
	hash2, err := hashstructure.Hash(dcBundle2, hashstructure.FormatV2, nil)
	if err != nil {
		return false, err
	}
	// CsvJSON and Objects are ignored by Hash, so they must be compared separately.
	return hash1 == hash2 && b1.CsvJSON == b2.CsvJSON && reflect.DeepEqual(b1.Objects, b2.Objects), nil
}

func addAllDependencies(newModel, oldModel, outputModel model.Model) error {
	// Get every oldModel's bundle's dependencies, and their dependencies, etc. by BFS.
	providingBundlesByPackage := map[string][]*model.Bundle{}
	for curr := getBundles(outputModel); len(curr) != 0; {
		reqGVKs, reqPkgs, err := findDependencies(curr)
		if err != nil {
			return err
		}
		// Break early so the entire source model is not iterated through unnecessarily.
		if len(reqGVKs) == 0 && len(reqPkgs) == 0 {
			break
		}
		curr = nil
		// Get bundles that provide dependencies from newModel, which should have
		// the latest bundles of each dependency package.
		for _, pkg := range newModel {
			providingBundles := getBundlesThatProvide(pkg, reqGVKs, reqPkgs)
			curr = append(curr, providingBundles...)

			oldPkg, oldHasPkg := oldModel[pkg.Name]
			for _, b := range providingBundles {
				// If the bundle is not in oldModel, add it to the set.
				// outputModel is checked below.
				add := true
				if oldHasPkg {
					if oldCh, oldHasCh := oldPkg.Channels[b.Channel.Name]; oldHasCh {
						_, oldHasBundle := oldCh.Bundles[b.Name]
						add = !oldHasBundle
					}
				}
				if add {
					providingBundlesByPackage[b.Package.Name] = append(providingBundlesByPackage[b.Package.Name], b)
				}
			}
		}
	}

	// Add the diff between an oldModel dependency package and its new counterpart
	// or the entire package if oldModel does not have it.
	for pkgName, bundles := range providingBundlesByPackage {
		newPkg := newModel[pkgName]
		heads := make(map[string]*model.Bundle, len(newPkg.Channels))
		for _, ch := range newPkg.Channels {
			var err error
			if heads[ch.Name], err = ch.Head(); err != nil {
				return err
			}
		}

		// Sort by version then channel so bundles lower in the full graph are more likely
		// to be included in previous loops.
		sort.Slice(bundles, func(i, j int) bool {
			if bundles[i].Channel.Name == bundles[j].Channel.Name {
				return bundles[i].Version.LT(bundles[j].Version)
			}
			return bundles[i].Channel.Name < bundles[j].Channel.Name
		})

		for _, b := range bundles {
			newCh := b.Channel

			// Continue if b was added in a previous loop iteration.
			// Otherwise create a new package/channel for b if they do not exist.
			var (
				outputPkg *model.Package
				outputCh  *model.Channel

				outHasPkg, outHasCh bool
			)
			if outputPkg, outHasPkg = outputModel[b.Package.Name]; outHasPkg {
				if outputCh, outHasCh = outputPkg.Channels[b.Channel.Name]; outHasCh {
					if _, outputHasBundle := outputCh.Bundles[b.Name]; outputHasBundle {
						continue
					}
				}
			} else {
				outputPkg = copyPackageNoChannels(newPkg)
				outputModel[outputPkg.Name] = outputPkg
			}
			if !outHasCh {
				outputCh = copyChannelNoBundles(newCh, outputPkg)
				outputPkg.Channels[outputCh.Name] = outputCh
			}

			head := heads[newCh.Name]
			graph := makeUpgradeGraph(newCh)
			intersectingBundles, intersectionFound := findIntersectingBundles(newCh, b, head, graph)
			if !intersectionFound {
				// This should never happen, since b and head are from the same model.
				return fmt.Errorf("channel %s: head %q not reachable from bundle %q", newCh.Name, head.Name, b.Name)
			}
			for _, ib := range intersectingBundles {
				if _, outHasBundle := outputCh.Bundles[ib.Name]; !outHasBundle {
					outputCh.Bundles[ib.Name] = copyBundle(ib, outputCh, outputPkg)
				}
			}
		}
	}

	return nil
}

// getBundles collects all bundles specified by m. Since each bundle
// references its package, their uniqueness property holds in a flat list.
func getBundles(m model.Model) (bundles []*model.Bundle) {
	for _, pkg := range m {
		for _, ch := range pkg.Channels {
			for _, b := range ch.Bundles {
				bundles = append(bundles, b)
			}
		}
	}
	return bundles
}

// findDependencies finds all GVK and package dependencies and indexes them
// by the apropriate key for lookups.
func findDependencies(bundles []*model.Bundle) (map[property.GVK]struct{}, map[string][]semver.Range, error) {
	// Find all dependencies of bundles in the output model.
	reqGVKs := map[property.GVK]struct{}{}
	reqPkgs := map[string][]semver.Range{}
	for _, b := range bundles {

		for _, gvkReq := range b.PropertiesP.GVKsRequired {
			gvk := property.GVK{
				Group:   gvkReq.Group,
				Version: gvkReq.Version,
				Kind:    gvkReq.Kind,
			}
			reqGVKs[gvk] = struct{}{}
		}

		for _, pkgReq := range b.PropertiesP.PackagesRequired {
			var inRange semver.Range
			if pkgReq.VersionRange != "" {
				var err error
				if inRange, err = semver.ParseRange(pkgReq.VersionRange); err != nil {
					// Should never happen since model has been validated.
					return nil, nil, err
				}
			} else {
				// Any bundle in this package will satisfy a range-less package requirement.
				inRange = func(semver.Version) bool { return true }
			}
			reqPkgs[pkgReq.PackageName] = append(reqPkgs[pkgReq.PackageName], inRange)
		}
	}

	return reqGVKs, reqPkgs, nil
}

// getBundlesThatProvide returns the latest-version bundles in pkg that provide
// a GVK or version in reqGVKs or reqPkgs, respectively.
func getBundlesThatProvide(pkg *model.Package, reqGVKs map[property.GVK]struct{}, reqPkgs map[string][]semver.Range) (providingBundles []*model.Bundle) {
	// Pre-allocate the amount of space needed for all ranges
	// specified by requiring bundles.
	var bundlesByRange [][]*model.Bundle
	ranges, isPkgRequired := reqPkgs[pkg.Name]
	if isPkgRequired {
		bundlesByRange = make([][]*model.Bundle, len(ranges))
	}
	// Collect package bundles that provide a GVK or are in a range.
	bundlesProvidingGVK := make(map[property.GVK][]*model.Bundle)
	for _, ch := range pkg.Channels {
		for _, b := range ch.Bundles {
			for _, gvk := range b.PropertiesP.GVKs {
				if _, hasGVK := reqGVKs[gvk]; hasGVK {
					bundlesProvidingGVK[gvk] = append(bundlesProvidingGVK[gvk], b)
				}
			}
			for i, inRange := range ranges {
				if inRange(b.Version) {
					bundlesByRange[i] = append(bundlesByRange[i], b)
				}
			}
		}
	}

	// Sort bundles providing a GVK by version and use the latest version.
	latestBundles := make(map[string]*model.Bundle)
	for gvk, bundles := range bundlesProvidingGVK {
		sort.Slice(bundles, func(i, j int) bool {
			return bundles[i].Version.LT(bundles[j].Version)
		})
		lb := bundles[len(bundles)-1]
		latestBundles[lb.Version.String()] = lb
		delete(reqGVKs, gvk)
	}

	// Sort bundles in a range by version and use the latest version.
	unsatisfiedRanges := []semver.Range{}
	for i, bundlesInRange := range bundlesByRange {
		if len(bundlesInRange) == 0 {
			unsatisfiedRanges = append(unsatisfiedRanges, ranges[i])
			continue
		}
		sort.Slice(bundlesInRange, func(i, j int) bool {
			return bundlesInRange[i].Version.LT(bundlesInRange[j].Version)
		})
		lb := bundlesInRange[len(bundlesInRange)-1]
		latestBundles[lb.Version.String()] = lb
	}
	if isPkgRequired && len(unsatisfiedRanges) == 0 {
		delete(reqPkgs, pkg.Name)
	}
	// TODO(estroz): handle missed ranges with logs.

	// Return deduplicated bundles that provide GVKs/versions.
	for _, b := range latestBundles {
		providingBundles = append(providingBundles, b)
	}
	return providingBundles
}

func convertFromModelBundle(b *model.Bundle) Bundle {
	return Bundle{
		Schema:        schemaBundle,
		Name:          b.Name,
		Package:       b.Package.Name,
		Image:         b.Image,
		RelatedImages: modelRelatedImagesToRelatedImages(b.RelatedImages),
		CsvJSON:       b.CsvJSON,
		Objects:       b.Objects,
		Properties:    b.Properties,
	}
}

func copyPackageNoChannels(in *model.Package) *model.Package {
	cp := &model.Package{
		Name:        in.Name,
		Description: in.Description,
		Channels:    make(map[string]*model.Channel, len(in.Channels)),
	}
	if in.Icon != nil {
		cp.Icon = &model.Icon{
			Data:      make([]byte, len(in.Icon.Data)),
			MediaType: in.Icon.MediaType,
		}
		copy(cp.Icon.Data, in.Icon.Data)
	}
	return cp
}

func copyPackage(in *model.Package) *model.Package {
	cp := copyPackageNoChannels(in)
	for _, ch := range in.Channels {
		cp.Channels[ch.Name] = copyChannel(ch, cp)
	}
	return cp
}

func copyChannelNoBundles(in *model.Channel, pkg *model.Package) *model.Channel {
	cp := &model.Channel{
		Name:    in.Name,
		Package: pkg,
		Bundles: make(map[string]*model.Bundle, len(in.Bundles)),
	}
	return cp
}

func copyChannel(in *model.Channel, pkg *model.Package) *model.Channel {
	cp := copyChannelNoBundles(in, pkg)
	for _, b := range in.Bundles {
		cp.Bundles[b.Name] = copyBundle(b, cp, pkg)
	}
	return cp
}

func copyBundle(in *model.Bundle, ch *model.Channel, pkg *model.Package) *model.Bundle {
	cp := &model.Bundle{
		Name:      in.Name,
		Channel:   ch,
		Package:   pkg,
		Image:     in.Image,
		Replaces:  in.Replaces,
		Version:   semver.MustParse(in.Version.String()),
		CsvJSON:   in.CsvJSON,
		SkipRange: in.SkipRange,
	}
	if in.PropertiesP != nil {
		cp.PropertiesP = new(property.Properties)
		*cp.PropertiesP = *in.PropertiesP
	}
	if len(in.Skips) != 0 {
		cp.Skips = make([]string, len(in.Skips))
		copy(cp.Skips, in.Skips)
	}
	if len(in.Properties) != 0 {
		cp.Properties = make([]property.Property, len(in.Properties))
		copy(cp.Properties, in.Properties)
	}
	if len(in.RelatedImages) != 0 {
		cp.RelatedImages = make([]model.RelatedImage, len(in.RelatedImages))
		copy(cp.RelatedImages, in.RelatedImages)
	}
	if len(in.Objects) != 0 {
		cp.Objects = make([]string, len(in.Objects))
		copy(cp.Objects, in.Objects)
	}
	return cp
}
