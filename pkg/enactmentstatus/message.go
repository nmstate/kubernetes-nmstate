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
	"bytes"
	"compress/gzip"
	b64 "encoding/base64"
	"io"
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
	if len(matches) == 2 { //nolint:gomnd
		sb.WriteString(strings.TrimRight(matches[1], " ") + "\n")
		index++
	}
	index = skipLines(errorLines, index)

	formatLines(index, errorLines, &sb)

	return sb.String()
}

// Loops over all lines in error message and selects lines that should be kept
func formatLines(index int, errorLines []string, sb *strings.Builder) {
	shouldFormatLine := true
	for index < len(errorLines) {
		if errorLines[index] == "---" {
			shouldFormatLine = false
		}
		processLine(&errorLines[index], sb, shouldFormatLine)
		index++
		index = skipLines(errorLines, index)
	}
}

// Simplifies a line that should be kept in the error message
func processLine(errorLine *string, sb *strings.Builder, shouldFormatLine bool) {
	if !shouldFormatLine {
		sb.WriteString(*errorLine + "\n")
		return
	}
	lineSplitByColon := strings.Split(strings.TrimRight(*errorLine, " "), ": ")
	indent := ""
	for _, lineSection := range lineSplitByColon {
		sb.WriteString(indent + lineSection + "\n")
		indent += "  "
	}
}

func skipLines(lines []string, index int) int {
	if index >= len(lines) {
		return index
	}

	regexMatchFileTrace := regexp.MustCompile(`\A\s*File "\S*",\sline\s\d*,\sin`)
	regexTraceback := regexp.MustCompile(`\A\s*Traceback\s\(most\srecent\scall\slast\):`)
	regexWithDateTime := regexp.MustCompile(`\A\s*\d{4}-\d{2}-\d{2}\s\d{2}:\d{2}:\d{2}`)
	regexCurrentState := regexp.MustCompile(`.*->\scurrentState:\s---`)
	regexUnhandledMessage := regexp.MustCompile(`\AUnhandled\s.*\sfor\s`)
	regexKeywords := regexp.MustCompile(`DEBUG|WARNING|UserWarning:|warnings\.warn\(`)

	trimmedLine := strings.Trim(lines[index], ": ")

	if lineEmpty(trimmedLine) {
		index++
		return skipLines(lines, index)
	}
	if matched := regexTraceback.MatchString(lines[index]); matched {
		index++
		isTraceback := regexMatchFileTrace.MatchString(lines[index])
		for isTraceback {
			index += 2
			isTraceback = regexMatchFileTrace.MatchString(lines[index])
		}
		return skipLines(lines, index)
	}
	if matched := regexWithDateTime.MatchString(lines[index]); matched {
		index++
		return skipLines(lines, index)
	}
	if matched := regexUnhandledMessage.MatchString(lines[index]); matched {
		index++
		return skipLines(lines, index)
	}
	if matched := regexKeywords.MatchString(lines[index]); matched {
		index++
		return skipLines(lines, index)
	}
	if matched := regexCurrentState.MatchString(lines[index]); matched {
		index = len(lines) - 1
		return skipLines(lines, index)
	}

	return index
}

func lineEmpty(trimmedLine string) bool {
	return trimmedLine == "" || trimmedLine == "'"
}

func CompressAndEncodeMessage(message string) string {
	compressedMessage, err := compressMessage(message)
	if err != nil {
		return ""
	}
	return encodeMessage(compressedMessage)
}

func compressMessage(message string) (bytes.Buffer, error) {
	var buf bytes.Buffer
	writer := gzip.NewWriter(&buf)

	_, err := writer.Write([]byte(message))
	if err != nil {
		return bytes.Buffer{}, err
	}

	if err := writer.Close(); err != nil {
		return bytes.Buffer{}, err
	}

	return buf, nil
}

func encodeMessage(buf bytes.Buffer) string {
	return b64.StdEncoding.EncodeToString(buf.Bytes())
}

func DecodeAndDecompressMessage(message string) string {
	decodedMessage := decodeMessage(message)
	return decompressMessage(decodedMessage)
}

func decodeMessage(encodedMessage string) []byte {
	data, _ := b64.StdEncoding.DecodeString(encodedMessage)
	return data
}

func decompressMessage(data []byte) string {
	bytesReader := bytes.NewReader(data)
	gzipReader, err := gzip.NewReader(bytesReader)
	if err != nil {
		return ""
	}
	decompressedMessage, _ := io.ReadAll(gzipReader)
	return string(decompressedMessage)
}
