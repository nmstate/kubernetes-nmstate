package shared

const (
	NmstateConditionAvailable   ConditionType = "Available"
	NmstateConditionDegraded    ConditionType = "Degraded"
	NmstateConditionProgressing ConditionType = "Progressing"

	NmstateInternalError       ConditionReason = "InternalError"
	NmstateApplyManifestsError ConditionReason = "ApplyManifestError"
	NmstateDeploying           ConditionReason = "Deploying"
)
