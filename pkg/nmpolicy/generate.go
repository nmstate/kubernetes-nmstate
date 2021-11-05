package nmpolicy

import (
	"github.com/nmstate/nmpolicy/nmpolicy"
	nmpolicytypes "github.com/nmstate/nmpolicy/nmpolicy/types"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nmstateapi "github.com/nmstate/kubernetes-nmstate/api/shared"
)

func GenerateState(desiredState nmstateapi.State,
	policySpec nmstateapi.NodeNetworkConfigurationPolicySpec,
	currentState nmstateapi.State,
	cachedState nmstateapi.NodeNetworkConfigurationEnactmentCachedState) (nmstateapi.NodeNetworkConfigurationEnactmentGeneratedState, error) {
	nmpolicySpec := nmpolicytypes.PolicySpec{
		Capture:      policySpec.Capture,
		DesiredState: []byte(desiredState.Raw),
	}
	nmpolicyGeneratedState, err := nmpolicy.GenerateState(nmpolicySpec, currentState.Raw, convertCachedStateFromEnactment(cachedState))
	if err != nil {
		return nmstateapi.NodeNetworkConfigurationEnactmentGeneratedState{}, err
	}

	return convertGeneratedStateToEnactment(nmpolicyGeneratedState), nil
}

func convertGeneratedStateToEnactment(nmpolicyGeneratedState nmpolicytypes.GeneratedState) nmstateapi.NodeNetworkConfigurationEnactmentGeneratedState {
	generatedState := nmstateapi.NodeNetworkConfigurationEnactmentGeneratedState{
		DesiredState: nmstateapi.State{
			Raw: nmpolicyGeneratedState.DesiredState,
		},
		MetaInfo: convertMetaInfoToEnactment(nmpolicyGeneratedState.MetaInfo),
	}

	for captureKey, capturedState := range nmpolicyGeneratedState.Cache.Capture {
		capturedState := nmstateapi.NodeNetworkConfigurationEnactmentCaptureState{
			State: nmstateapi.State{
				Raw: capturedState.State,
			},
			MetaInfo: convertMetaInfoToEnactment(capturedState.MetaInfo),
		}
		generatedState.Cache.Capture[captureKey] = capturedState
	}
	return generatedState
}

func convertCachedStateFromEnactment(enactmentCachedState nmstateapi.NodeNetworkConfigurationEnactmentCachedState) nmpolicytypes.CachedState {
	cachedState := nmpolicytypes.CachedState{}
	for captureKey, capturedState := range enactmentCachedState.Capture {
		capturedState := nmpolicytypes.CaptureState{
			State: capturedState.State.Raw,
			MetaInfo: nmpolicytypes.MetaInfo{
				Version:   capturedState.MetaInfo.Version,
				TimeStamp: capturedState.MetaInfo.TimeStamp.Time,
			},
		}
		cachedState.Capture[captureKey] = capturedState
	}
	return cachedState
}

func convertMetaInfoToEnactment(metaInfo nmpolicytypes.MetaInfo) nmstateapi.NodeNetworkConfigurationEnactmentMetaInfo {
	return nmstateapi.NodeNetworkConfigurationEnactmentMetaInfo{
		Version:   metaInfo.Version,
		TimeStamp: metav1.NewTime(metaInfo.TimeStamp),
	}
}
