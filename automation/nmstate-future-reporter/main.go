package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	basicProwURL        = "https://storage.googleapis.com/kubevirt-prow/logs/periodic-knmstate-e2e-handler-k8s-future"
	latestBuildURL      = basicProwURL + "/latest-build.txt"
	finishedURLTemplate = basicProwURL + "/%s/finished.json"
	jobURLTemplate      = basicProwURL + "/%s/prowjob.json"
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

var (
	webhookURL string
	fakeReport *string
	dryRun     *bool
)

func init() {
	fakeReport = flag.String("fake", "", "Generate a fake report (use 'success', 'failure', or 'stale')")
	dryRun = flag.Bool("dry-run", false, "Print the message that would be sent without actually sending it")
	flag.Parse()

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
	message, err := generateMessage()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to generate message: %v\n", err)
		os.Exit(1)
	}

	err = sendMessageToSlackChannel(message)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to send message to slack channel: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Message sent successfully")
}

func generateMessage() (slackMessage, error) {
	var buildId string
	var buildStatus finished
	var jobURL string
	var err error

	if *fakeReport != "" {
		buildId, buildStatus, jobURL, err = generateFakeData(*fakeReport)
		if err != nil {
			return slackMessage{}, fmt.Errorf("failed to generate fake data: %w", err)
		}
	} else {
		buildId, err = getLatestBuild()
		if err != nil {
			return slackMessage{}, fmt.Errorf("failed to get latest build: %w", err)
		}

		buildStatus, err = getBuildStatus(buildId)
		if err != nil {
			return slackMessage{}, fmt.Errorf("failed to get build status: %w", err)
		}

		jobURL, err = getJob(buildId)
		if err != nil {
			return slackMessage{}, fmt.Errorf("failed to get job URL: %w", err)
		}
	}

	message := generateStatusMessage(buildStatus, jobURL, buildId)
	return message, nil
}

func generateFakeData(reportType string) (string, finished, string, error) {
	buildId := "1234567890"
	jobURL := "https://prow.ci.kubevirt.io/view/gs/kubevirt-prow/logs/periodic-knmstate-e2e-handler-k8s-future/1234567890"

	var buildStatus finished
	switch reportType {
	case "success":
		buildStatus = finished{
			Timestamp: time.Now().Unix(),
			Passed:    true,
			Result:    "SUCCESS",
			Revision:  "abc123def456",
		}
	case "failure":
		buildStatus = finished{
			Timestamp: time.Now().Unix(),
			Passed:    false,
			Result:    "FAILURE",
			Revision:  "abc123def456",
		}
	case "stale":
		buildStatus = finished{
			Timestamp: time.Now().Add(-48 * time.Hour).Unix(),
			Passed:    false,
			Result:    "FAILURE",
			Revision:  "abc123def456",
		}
	default:
		return "", finished{}, "", fmt.Errorf("invalid fake report type: %s (use 'success', 'failure', or 'stale')", reportType)
	}

	return buildId, buildStatus, jobURL, nil
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

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payload))
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

func generateStatusMessage(buildStatus finished, jobURL, buildId string) slackMessage {
	buildTime := buildStatus.getBuildTime()
	now := time.Now().UTC()

	var messageText string

	// Check if the build is older than 24 hours
	if now.Sub(buildTime) > 24*time.Hour {
		messageText = fmt.Sprintf(":warning: Nightly e2e for <https://copr.fedorainfracloud.org/coprs/nmstate/nmstate-git/|nmstate-git>: No build in last 24h")
	} else {
		var statusEmoji string
		if buildStatus.Passed {
			statusEmoji = ":solid-success:"
		} else {
			statusEmoji = ":failed:"
		}

		messageText = fmt.Sprintf("%s Nightly e2e for <https://copr.fedorainfracloud.org/coprs/nmstate/nmstate-git/|nmstate-git>: <%s|*%s*>",
			statusEmoji,
			jobURL,
			strings.ToLower(buildStatus.Result))
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
	resp, err := http.Get(latestBuildURL)
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

func getBuildStatus(buildId string) (finished, error) {
	url := fmt.Sprintf(finishedURLTemplate, buildId)
	resp, err := http.Get(url)
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

func getJob(buildId string) (string, error) {
	url := fmt.Sprintf(jobURLTemplate, buildId)
	resp, err := http.Get(url)
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
