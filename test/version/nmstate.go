package version

import (
	"fmt"
	"regexp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/Masterminds/semver"

	"github.com/nmstate/kubernetes-nmstate/test/runner"
)

var (
	r = regexp.MustCompile(`nmstate-(.*)-.*`)
)

func IsNmstate(constraint string) bool {
	rpmVersion := runner.RunAtFirstHandlerPod("rpm", "-qe", "nmstate")

	By(fmt.Sprintf("Parsing nmstate version from rpm version %s", rpmVersion))
	submatchResult := r.FindStringSubmatch(rpmVersion)
	ExpectWithOffset(1, submatchResult).To(HaveLen(2))

	nmstateVersion := r.FindStringSubmatch(rpmVersion)[1]
	c, err := semver.NewConstraint(constraint)
	Expect(err).ToNot(HaveOccurred())

	v, err := semver.NewVersion(nmstateVersion)
	Expect(err).ToNot(HaveOccurred())

	By(fmt.Sprintf("Checking if nmstate version '%s' is '%s'", nmstateVersion, constraint))
	return c.Check(v)
}
