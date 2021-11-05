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
	cachedState map[string]nmstateapi.NodeNetworkConfigurationEnactmentCapturedState) (map[string]nmstateapi.NodeNetworkConfigurationEnactmentCapturedState,
	nmstateapi.NodeNetworkConfigurationEnactmentMetaInfo, nmstateapi.State, error) {
	nmpolicySpec := nmpolicytypes.PolicySpec{
		Capture:      policySpec.Capture,
		DesiredState: []byte(desiredState.Raw),
	}
	nmpolicyGeneratedState, err := nmpolicy.GenerateState(nmpolicySpec, currentState.Raw, convertCachedStateFromEnactment(cachedState))
	if err != nil {
		return map[string]nmstateapi.NodeNetworkConfigurationEnactmentCapturedState{}, nmstateapi.NodeNetworkConfigurationEnactmentMetaInfo{}, nmstateapi.State{}, err
	}

	capturedStates, desriedStateMetaInfo, desiredState := convertGeneratedStateToEnactmentConfig(nmpolicyGeneratedState)
	return capturedStates, desriedStateMetaInfo, desiredState, nil
}

func convertGeneratedStateToEnactmentConfig(nmpolicyGeneratedState nmpolicytypes.GeneratedState) (map[string]nmstateapi.NodeNetworkConfigurationEnactmentCapturedState,
	nmstateapi.NodeNetworkConfigurationEnactmentMetaInfo,
	nmstateapi.State) {
	desiredStateMetaInfo := convertMetaInfoToEnactment(nmpolicyGeneratedState.MetaInfo)
	capturedStates := make(map[string]nmstateapi.NodeNetworkConfigurationEnactmentCapturedState)

	for captureKey, capturedState := range nmpolicyGeneratedState.Cache.Capture {
		capturedState := nmstateapi.NodeNetworkConfigurationEnactmentCapturedState{
			State: nmstateapi.State{
				Raw: capturedState.State,
			},
			MetaInfo: convertMetaInfoToEnactment(capturedState.MetaInfo),
		}
		capturedStates[captureKey] = capturedState
	}
	return capturedStates, desiredStateMetaInfo, nmstateapi.State{Raw: nmpolicyGeneratedState.DesiredState}
}

func convertCachedStateFromEnactment(enactmentCachedState map[string]nmstateapi.NodeNetworkConfigurationEnactmentCapturedState) nmpolicytypes.CachedState {
	cachedState := nmpolicytypes.CachedState{Capture: make(map[string]nmpolicytypes.CaptureState)}
	for captureKey, capturedState := range enactmentCachedState {
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
