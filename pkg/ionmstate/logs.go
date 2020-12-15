package ionmstate

import (
	"encoding/json"
	"fmt"
)

func ConvertLogsToString(ll []Logs) string {
	logsAsBytes, err := json.MarshalIndent(&ll, "", "    ")
	if err != nil {
		return fmt.Sprintf("%+v", ll)
	}
	return string(logsAsBytes)
}
