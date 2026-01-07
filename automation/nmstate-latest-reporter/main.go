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

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	basicProwURL        = "https://storage.googleapis.com/kubevirt-prow/logs/periodic-knmstate-e2e-handler-k8s-latest"
	latestBuildURL      = basicProwURL + "/latest-build.txt"
	finishedURLTemplate = basicProwURL + "/%s/finished.json"
	jobURLTemplate      = basicProwURL + "/%s/prowjob.json"
	buildLogURLTemplate = basicProwURL + "/%s/build-log.txt"

	// minRegexMatches is the minimum number of regex matches expected (including the full match)
	minRegexMatches = 2

	// httpTimeout is the timeout duration for HTTP requests
	httpTimeout = 30 * time.Second
)

var (
	// nmstateVersionRe matches nmstate version strings in build logs
	// Example: nmstate-2.2.55-0.20251031.2666git045ed3c4.el9
	nmstateVersionRe = regexp.MustCompile(`nmstate-(\d+\.\d+\.\d+[-\w.]+\.el\d+)`)

	// networkManagerVersionRe matches NetworkManager version strings in build logs
	// Example: NetworkManager-1:1.55.4-34245.copr.5f85b55f7f.el9.x86_64
	networkManagerVersionRe = regexp.MustCompile(`NetworkManager-1:(\d+\.\d+\.\d+[-\w.]+\.el\d+)`)

	// elVersionSuffixRe matches the Enterprise Linux version suffix (e.g., .el9, .el10)
	elVersionSuffixRe = regexp.MustCompile(`\.el\d+$`)
)

type finished struct {
	Timestamp int64  `json:"timestamp"`
	Passed    bool   `json:"passed"`
	Result    string `json:"result"`
	Revision  string `json:"revision"`
}

func (f finished) getBuildTime() time.Time {
	return time.Unix(f.Timestamp, 0).UTC()
}

type prowJob struct {
	Status struct {
		URL string `json:"url"`
	} `json:"status"`
}

type versions struct {
	Nmstate               string
	NmstateBuildID        string
	NetworkManager        string
	NetworkManagerBuildID string
}

type slackMessage struct {
	Blocks []slackBlock `json:"blocks"`
}

type slackBlock struct {
	Type string           `json:"type"`
	Text *slackTextObject `json:"text,omitempty"`
}

type slackTextObject struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// stringSlice implements flag.Value for collecting multiple flag values
type stringSlice []string

func (s *stringSlice) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSlice) Set(value string) error {
	*s = append(*s, value)
	return nil
}

var (
	webhookURL string
	fakeReport *string
	dryRun     *bool
	notifyOn   stringSlice
)

func init() {
	fakeReport = flag.String("fake", "", "Generate a fake report (use 'success', 'failure', or 'stale')")
	dryRun = flag.Bool("dry-run", false, "Print the message that would be sent without actually sending it")
	flag.Var(&notifyOn, "notify-on", "Events to notify on (can be specified multiple times: success, failure, stale). Default: all events")
	flag.Parse()

	// Default to all events if none specified
	if len(notifyOn) == 0 {
		notifyOn = stringSlice{"success", "failure", "stale"}
	}

	// Skip env var validation in dry-run mode
	if *dryRun {
		return
	}

	var ok bool
	webhookURL, ok = os.LookupEnv("NMSTATE_SLACK_WEBHOOK_URL")
	if !ok {
		fmt.Fprintln(os.Stderr, "NMSTATE_SLACK_WEBHOOK_URL environment variable not set")
		os.Exit(1)
	}
}

func main() {
	message, buildStatus, err := generateMessage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate message: %v\n", err)
		os.Exit(1)
	}

	// Determine event type
	eventType := getEventType(buildStatus)

	// Check if we should notify for this event type
	if !shouldNotify(eventType) {
		fmt.Printf("Skipping notification for event type '%s' (not in notify-on list: %v)\n", eventType, notifyOn)
		return
	}

	err = sendMessageToSlackChannel(message)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to send message to slack channel: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Message sent successfully")
}

func getEventType(buildStatus finished) string {
	buildTime := buildStatus.getBuildTime()
	now := time.Now().UTC()

	// Check if the build is older than 24 hours
	if now.Sub(buildTime) > 24*time.Hour {
		return "stale"
	}

	if buildStatus.Passed {
		return "success"
	}

	return "failure"
}

func shouldNotify(eventType string) bool {
	for _, event := range notifyOn {
		if event == eventType {
			return true
		}
	}
	return false
}

func generateMessage() (slackMessage, finished, error) {
	var buildID string
	var buildStatus finished
	var jobURL string
	var vers versions
	var err error

	if *fakeReport != "" {
		_, buildStatus, jobURL, err = generateFakeData(*fakeReport)
		if err != nil {
			return slackMessage{}, finished{}, fmt.Errorf("failed to generate fake data: %w", err)
		}
		vers = versions{
			Nmstate:               "2.2.55-0.20251031.2666git045ed3c4.fake.el9",
			NmstateBuildID:        "9752008.fake",
			NetworkManager:        "1.55.4-34245.copr.5f85b55f7f.fake.el9",
			NetworkManagerBuildID: "9760340.fake",
		}
	} else {
		buildID, err = getLatestBuild()
		if err != nil {
			return slackMessage{}, finished{}, fmt.Errorf("failed to get latest build: %w", err)
		}

		buildStatus, err = getBuildStatus(buildID)
		if err != nil {
			return slackMessage{}, finished{}, fmt.Errorf("failed to get build status: %w", err)
		}

		jobURL, err = getJob(buildID)
		if err != nil {
			return slackMessage{}, finished{}, fmt.Errorf("failed to get job URL: %w", err)
		}

		vers, err = getVersions(buildID)
		if err != nil {
			return slackMessage{}, finished{}, fmt.Errorf("failed to get versions: %w", err)
		}
	}

	message := generateStatusMessage(buildStatus, jobURL, vers)
	return message, buildStatus, nil
}

func generateFakeData(reportType string) (buildID string, status finished, url string, err error) {
	buildID = "1234567890"
	url = "https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/logs/periodic-knmstate-e2e-handler-k8s-latest/1234567890"

	switch reportType {
	case "success":
		status = finished{
			Timestamp: time.Now().Unix(),
			Passed:    true,
			Result:    "SUCCESS",
			Revision:  "abc123def456",
		}
	case "failure":
		status = finished{
			Timestamp: time.Now().Unix(),
			Passed:    false,
			Result:    "FAILURE",
			Revision:  "abc123def456",
		}
	case "stale":
		status = finished{
			Timestamp: time.Now().Add(-48 * time.Hour).Unix(),
			Passed:    false,
			Result:    "FAILURE",
			Revision:  "abc123def456",
		}
	default:
		return "", finished{}, "", fmt.Errorf("invalid fake report type: %s (use 'success', 'failure', or 'stale')", reportType)
	}

	return buildID, status, url, nil
}

func sendMessageToSlackChannel(message slackMessage) error {
	if *dryRun {
		fmt.Println("=== DRY RUN MODE - Message Preview ===")
		messageJSON, err := json.MarshalIndent(message, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal message to JSON: %w", err)
		}
		fmt.Println(string(messageJSON))
		fmt.Println("\n=== Formatted Message ===")
		printFormattedMessage(message.Blocks)
		return nil
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create webhook request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func printFormattedMessage(blocks []slackBlock) {
	for _, block := range blocks {
		if block.Text != nil {
			fmt.Println(block.Text.Text)
		}
	}
}

func generateStatusMessage(buildStatus finished, jobURL string, vers versions) slackMessage {
	buildTime := buildStatus.getBuildTime()
	now := time.Now().UTC()

	var messageText string

	// Generate nmstate link with build ID if available
	nmstateLink := "https://copr.fedorainfracloud.org/coprs/nmstate/nmstate-git/"
	if vers.NmstateBuildID != "" {
		nmstateLink = fmt.Sprintf("https://copr.fedorainfracloud.org/coprs/nmstate/nmstate-git/build/%s/", vers.NmstateBuildID)
	}

	// Generate NetworkManager link with build ID if available
	nmLink := "https://copr.fedorainfracloud.org/coprs/networkmanager/NetworkManager-main/"
	if vers.NetworkManagerBuildID != "" {
		nmLink = fmt.Sprintf("https://copr.fedorainfracloud.org/coprs/networkmanager/NetworkManager-main/build/%s/", vers.NetworkManagerBuildID)
	}

	// Check if the build is older than 24 hours
	if now.Sub(buildTime) > 24*time.Hour {
		messageText = fmt.Sprintf(":warning: Nightly e2e: <%s|No build in last 24h>", jobURL)
	} else {
		var statusEmoji string
		if buildStatus.Passed {
			statusEmoji = ":solid-success:"
		} else {
			statusEmoji = ":failed:"
		}

		messageText = fmt.Sprintf("%s Nightly e2e: <%s|*%s*>\n• <%s|nmstate %s>\n• <%s|NetworkManager %s>",
			statusEmoji,
			jobURL,
			strings.ToLower(buildStatus.Result),
			nmstateLink,
			vers.Nmstate,
			nmLink,
			vers.NetworkManager)
	}

	return slackMessage{
		Blocks: []slackBlock{
			{
				Type: "section",
				Text: &slackTextObject{
					Type: "mrkdwn",
					Text: messageText,
				},
			},
		},
	}
}

func getLatestBuild() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestBuildURL, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch latest build: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

func getBuildStatus(buildID string) (finished, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()
	url := fmt.Sprintf(finishedURLTemplate, buildID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return finished{}, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return finished{}, fmt.Errorf("failed to fetch build status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return finished{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var status finished
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return finished{}, fmt.Errorf("failed to decode response: %w", err)
	}

	return status, nil
}

func getJob(buildID string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()
	url := fmt.Sprintf(jobURLTemplate, buildID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch job: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var job prowJob
	if err := json.NewDecoder(resp.Body).Decode(&job); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	return job.Status.URL, nil
}

func getVersions(buildID string) (versions, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()
	url := fmt.Sprintf(buildLogURLTemplate, buildID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return versions{}, fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return versions{}, fmt.Errorf("failed to fetch build log: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return versions{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return versions{}, fmt.Errorf("failed to read build log: %w", err)
	}

	bodyStr := string(body)
	vers := versions{
		Nmstate:               "unknown",
		NmstateBuildID:        "",
		NetworkManager:        "unknown",
		NetworkManagerBuildID: "",
	}

	// Look for nmstate version pattern: nmstate-X.Y.Z-...
	// Example: nmstate-2.2.55-0.20251031.2666git045ed3c4.el9
	if matches := nmstateVersionRe.FindStringSubmatch(bodyStr); len(matches) >= minRegexMatches {
		vers.Nmstate = matches[1]
		// Get the copr build ID for this version

		buildID, err := getCoprBuildID("nmstate", "nmstate-git", "nmstate", vers.Nmstate)
		if err == nil {
			vers.NmstateBuildID = buildID
		}
	}

	// Look for NetworkManager version pattern in upgrade lines
	// Example from copr: NetworkManager-1:1.55.4-34245.copr.5f85b55f7f.el9.x86_64
	// Example from regular repo: NetworkManager-1:1.54.1-1.el9.x86_64
	if matches := networkManagerVersionRe.FindStringSubmatch(bodyStr); len(matches) >= minRegexMatches {
		vers.NetworkManager = matches[1]
		// Try to get copr build ID for this version
		buildID, err := getCoprBuildID("networkmanager", "NetworkManager-main", "NetworkManager", vers.NetworkManager)
		if err == nil {
			vers.NetworkManagerBuildID = buildID
		}
	}

	return vers, nil
}

type coprBuildList struct {
	Items []coprBuild `json:"items"`
}

type coprBuild struct {
	ID            int               `json:"id"`
	SourcePackage coprSourcePackage `json:"source_package"`
}

type coprSourcePackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func getCoprBuildID(ownerName, projectName, packageName, packageVersion string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), httpTimeout)
	defer cancel()
	url := fmt.Sprintf(
		"https://copr.fedorainfracloud.org/api_3/build/list?ownername=%s&projectname=%s&packagename=%s",
		ownerName, projectName, packageName,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch copr builds: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code from copr API: %d", resp.StatusCode)
	}

	var buildList coprBuildList
	if err := json.NewDecoder(resp.Body).Decode(&buildList); err != nil {
		return "", fmt.Errorf("failed to decode copr API response: %w", err)
	}

	// The version in the build log includes .elN suffix (e.g., .el9, .el10), but the copr API version doesn't
	// Strip the .elN suffix to match
	versionToMatch := elVersionSuffixRe.ReplaceAllString(packageVersion, "")

	// Search for the build with matching version
	for _, build := range buildList.Items {
		if strings.HasPrefix(versionToMatch, build.SourcePackage.Version+"-") || versionToMatch == build.SourcePackage.Version {
			return fmt.Sprintf("%d", build.ID), nil
		}
	}

	return "", fmt.Errorf("no copr build found for %q version %s", projectName, packageVersion)
}
