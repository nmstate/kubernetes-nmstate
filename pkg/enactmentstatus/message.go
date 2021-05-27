/*
Copyright The Kubernetes NMState Authors.


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package enactmentstatus

import (
	"regexp"
	"strings"
)

// The function strips disarranged parts of the error messages, so it is easier to read for the user
func FormatErrorString(errorMessage string) string {
	var sb strings.Builder
	errorLines := strings.Split(errorMessage, "\n")
	index := 0
	if strings.Contains(errorLines[0], "error reconciling NodeNetworkConfigurationPolicy") {
		sb.WriteString(errorLines[0] + "\n")
		index++
	}

	re := regexp.MustCompile(`\s*(failed to execute.*'exit status \d+') '{2}\s'`)
	matches := re.FindStringSubmatch(errorLines[index])
	if len(matches) == 2 {
		sb.WriteString(strings.TrimRight(matches[1], " ") + "\n")
		index++
	}
	index = skipLines(errorLines, index)

	formatLines(index, errorLines, &sb)

	return sb.String()
}

//Loops over all lines in error message and selects lines that should be kept
func formatLines(index int, errorLines []string, sb *strings.Builder) {
	for index < len(errorLines) {
		formatLine(&errorLines[index], sb)
		index++
		index = skipLines(errorLines, index)
	}
}

//Simplifies a line that should be kept in the error message
func formatLine(errorLine *string, sb *strings.Builder) {
	lineSplitByColon := strings.Split(strings.TrimRight(*errorLine, " "), ": ")
	indent := ""
	for _, lineSection := range lineSplitByColon {
		sb.WriteString(indent + lineSection + "\n")
		indent = indent + "  "
	}
}

func skipLines(lines []string, index int) int {
	if index >= len(lines) {
		return index
	}

	regexMatchFileTrace := `\A\s*File "\S*",\sline\s\d*,\sin`
	regexTraceback := `\A\s*Traceback\s\(most\srecent\scall\slast\):`
	regexWithDateTime := `\A\s*\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}:\d{2}`
	regexCurrentState := `.*->\scurrentState:\s---`
	regexUnhandledMessage := `\AUnhandled\s.*\sfor\s`
	regexKeywords := `DEBUG|WARNING|UserWarning:|warnings\.warn\(`

	trimmedLine := strings.Trim(lines[index], ": ")

	if lineEmpty(trimmedLine) {
		index++
		return skipLines(lines, index)
	}
	if matched, _ := regexp.MatchString(regexTraceback, lines[index]); matched == true {
		index++
		isTraceback, _ := regexp.MatchString(regexMatchFileTrace, lines[index])
		for isTraceback {
			index += 2
			isTraceback, _ = regexp.MatchString(regexMatchFileTrace, lines[index])
		}
		return skipLines(lines, index)
	}
	if matched, _ := regexp.MatchString(regexWithDateTime, lines[index]); matched == true {
		index++
		return skipLines(lines, index)
	}
	if matched, _ := regexp.MatchString(regexUnhandledMessage, lines[index]); matched == true {
		index++
		return skipLines(lines, index)
	}
	if matched, _ := regexp.MatchString(regexKeywords, lines[index]); matched == true {
		index++
		return skipLines(lines, index)
	}
	if matched, _ := regexp.MatchString(regexCurrentState, lines[index]); matched == true {
		index = len(lines) - 1
		return skipLines(lines, index)
	}

	return index
}

func lineEmpty(trimmedLine string) bool {
	return trimmedLine == "" || trimmedLine == "'"
}
