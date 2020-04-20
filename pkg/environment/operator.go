package environment

import "os"

// IsOperator returns true when RUN_OPERATOR env var is present
func IsOperator() bool {
	_, runOperator := os.LookupEnv("RUN_OPERATOR")
	return runOperator
}

// IsOperator returns true when RUN_OPERATOR env var is present
func IsWebhookServer() bool {
	_, runWebhookServer := os.LookupEnv("RUN_WEBHOOK_SERVER")
	return runWebhookServer
}

// IsOperator returns true when RUN_OPERATOR env var is present
func OperatorName() string {
	return os.Getenv("OPERATOR_NAME")
}
