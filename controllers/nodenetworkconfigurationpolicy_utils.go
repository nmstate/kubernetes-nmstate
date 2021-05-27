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

package controllers

import (
	"bytes"
	"compress/flate"
	b64 "encoding/base64"
	"io/ioutil"
	"regexp"
	"strings"
)

// The function strips disarranged parts of the error messages, so it is easier to read for the user
func formatErrorString(err error) string {
	var sb strings.Builder
	errorMessage := err.Error()
	errorLines := strings.Split(errorMessage, "\n")
	index := 0

	re := regexp.MustCompile(`\s*(.*--no-commit --timeout \d+: 'exit status \d+') '{2}\s'`)
	matches := re.FindStringSubmatch(errorLines[index])
	if len(matches) == 2 {
		sb.WriteString(strings.TrimRight(matches[1], " ") + "\n")
		index++
	}
	index = skipLines(&errorLines, index)

	for index < len(errorLines) {
		lineSplitByColon := strings.Split(strings.TrimRight(errorLines[index], " "), ": ")
		indent := ""
		for _, lineSection := range lineSplitByColon {
			sb.WriteString(indent + lineSection + "\n")
			indent = indent + "  "
		}
		index++
		index = skipLines(&errorLines, index)
	}

	return sb.String()
}

func skipLines(lines *[]string, index int) int {
	if index >= len(*lines) {
		return index
	}

	regexMatchFileTrace := `\A\s*File "\S*",\sline\s\d*,\sin`
	regexTraceback := `\A\s*Traceback\s\(most\srecent\scall\slast\):`
	regexWithDateTime := `\A\s*\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}:\d{2}`
	regexCurrentState := `.*->\scurrentState:\s---`
	regexUnhandledMessage := `\AUnhandled\s.*\sfor\s`
	regexKeywords := `DEBUG|WARNING|UserWarning:|warnings\.warn\(`

	trimmedLine := strings.Trim((*lines)[index], ": ")

	if trimmedLine == "" || trimmedLine == "'" {
		index++
		return skipLines(lines, index)
	}
	if matched, _ := regexp.MatchString(regexTraceback, (*lines)[index]); matched == true {
		index++
		isTraceback, _ := regexp.MatchString(regexMatchFileTrace, (*lines)[index])
		for isTraceback {
			index += 2
			isTraceback, _ = regexp.MatchString(regexMatchFileTrace, (*lines)[index])
		}
		return skipLines(lines, index)
	}
	if matched, _ := regexp.MatchString(regexWithDateTime, (*lines)[index]); matched == true {
		index++
		return skipLines(lines, index)
	}
	if matched, _ := regexp.MatchString(regexUnhandledMessage, (*lines)[index]); matched == true {
		index++
		return skipLines(lines, index)
	}
	if matched, _ := regexp.MatchString(regexKeywords, (*lines)[index]); matched == true {
		index++
		return skipLines(lines, index)
	}
	if matched, _ := regexp.MatchString(regexCurrentState, (*lines)[index]); matched == true {
		index = len(*lines) - 1
		return skipLines(lines, index)
	}

	return index
}

func encodeMessage(message string) (string, error) {
	var buf bytes.Buffer
	writer, err := flate.NewWriter(&buf, flate.BestCompression)

	if err != nil {
		return "", err
	}

	_, err = writer.Write([]byte(message))
	if err != nil {
		return "", err
	}

	if err := writer.Close(); err != nil {
		return "", err
	}

	return b64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func decodeMessage(encodedMessage string) string {
	data, _ := b64.StdEncoding.DecodeString(encodedMessage)
	bytesReader := bytes.NewReader(data)
	flateReader := flate.NewReader(bytesReader)
	decodedMessage, _ := ioutil.ReadAll(flateReader)
	return string(decodedMessage)
}
