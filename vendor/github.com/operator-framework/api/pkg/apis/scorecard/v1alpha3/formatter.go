package v1alpha3

import (
	"bufio"
	"fmt"
	"strings"
)

func (s Test) MarshalText() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("%s\n", strings.Repeat("-", 80)))
	sb.WriteString(fmt.Sprintf("Image:      %s\n", s.Spec.Image))

	if len(s.Spec.Entrypoint) > 0 {
		sb.WriteString(fmt.Sprintf("Entrypoint: %s\n", s.Spec.Entrypoint))
	}

	if len(s.Spec.Labels) > 0 {
		sb.WriteString("Labels:\n")
		for labelKey, labelValue := range s.Spec.Labels {
			sb.WriteString(fmt.Sprintf("\t%q:%q\n", labelKey, labelValue))
		}
	}
	if len(s.Status.Results) > 0 {
		sb.WriteString("Results:\n")
		for _, result := range s.Status.Results {
			if len(result.Name) > 0 {
				sb.WriteString(fmt.Sprintf("\tName: %s\n", result.Name))
			}
			sb.WriteString("\tState: ")
			switch result.State {
			case PassState, FailState, ErrorState:
				sb.WriteString(string(result.State))
				sb.WriteString("\n")
			default:
				sb.WriteString("unknown")
			}
			sb.WriteString("\n")

			if len(result.Suggestions) > 0 {
				sb.WriteString("\tSuggestions:\n")
				for _, suggestion := range result.Suggestions {
					sb.WriteString(fmt.Sprintf("\t\t%s\n", suggestion))
				}
			}

			if len(result.Errors) > 0 {
				sb.WriteString("\tErrors:\n")
				for _, err := range result.Errors {
					sb.WriteString(fmt.Sprintf("\t\t%s\n", err))
				}
			}

			if result.Log != "" {
				sb.WriteString("\tLog:\n")
				scanner := bufio.NewScanner(strings.NewReader(result.Log))
				for scanner.Scan() {
					sb.WriteString(fmt.Sprintf("\t\t%s\n", scanner.Text()))
				}
			}
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
