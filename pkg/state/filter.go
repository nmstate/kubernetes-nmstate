package state

import (
	"regexp"

	"github.com/nmstate/kubernetes-nmstate/api/shared"
)

var (
	gcTimerRexp = regexp.MustCompile(` *gc-timer: *[0-9]*\n`)
)

func RemoveDynamicAttributes(state string) string {

	// Remove attributes that make network state always different
	return gcTimerRexp.ReplaceAllLiteralString(state, "")
}

func RemoveDynamicAttributesFromStruct(state shared.State) string {
	return RemoveDynamicAttributes(state.String())
}
