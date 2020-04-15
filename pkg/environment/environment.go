package environment

import "os"

// IsOperator returns true when RUN_OPERATOR env var is present
func IsOperator() bool {
	_, runOperator := os.LookupEnv("RUN_OPERATOR")
	return runOperator
}

// IsWebhook returns true when RUN_WEBHOOK_SERVER env var is present
func IsWebhook() bool {
	_, runOperator := os.LookupEnv("RUN_WEBHOOK_SERVER")
	return runOperator
}

// IsHandler returns true if it's not the operator or webhook server
func IsHandler() bool {
	return !IsWebhook() && !IsOperator()
}

// Returns node name runnig the pod
func NodeName() string {
	return os.Getenv("NODE_NAME")
}
