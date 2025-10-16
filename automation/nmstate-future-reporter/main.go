package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/slack-go/slack"
)

const (
	basicProwURL       = "https://storage.googleapis.com/kubevirt-prow/logs/periodic-knmstate-e2e-handler-k8s-future"
	latestBuildURL     = basicProwURL + "/latest-build.txt"
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

var (
	token       string
	channelId   string
	fakeReport  *string
	dryRun      *bool
)

func init() {
	fakeReport = flag.String("fake", "", "Generate a fake report (use 'success' or 'failure')")
	dryRun = flag.Bool("dry-run", false, "Print the message that would be sent without actually sending it")
	flag.Parse()

	// Skip env var validation in dry-run mode
	if *dryRun {
		return
	}

	var ok bool
	token, ok = os.LookupEnv("NMSTATE_REPORTER_SLACK_TOKEN")
	if !ok {
		fmt.Fprintln(os.Stderr, "NMSTATE_REPORTER_SLACK_TOKEN environment variable not set")
		os.Exit(1)
	}

	channelId, ok = os.LookupEnv("NMSTATE_CHANNEL_ID")
	if !ok {
		fmt.Fprintln(os.Stderr, "NMSTATE_CHANNEL_ID environment variable not set")
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

func generateMessage() ([]slack.Block, error) {
	var buildId string
	var buildStatus finished
	var jobURL string
	var err error

	if *fakeReport != "" {
		buildId, buildStatus, jobURL, err = generateFakeData(*fakeReport)
		if err != nil {
			return nil, fmt.Errorf("failed to generate fake data: %w", err)
		}
	} else {
		buildId, err = getLatestBuild()
		if err != nil {
			return nil, fmt.Errorf("failed to get latest build: %w", err)
		}

		buildStatus, err = getBuildStatus(buildId)
		if err != nil {
			return nil, fmt.Errorf("failed to get build status: %w", err)
		}

		jobURL, err = getJob(buildId)
		if err != nil {
			return nil, fmt.Errorf("failed to get job URL: %w", err)
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
	default:
		return "", finished{}, "", fmt.Errorf("invalid fake report type: %s (use 'success' or 'failure')", reportType)
	}

	return buildId, buildStatus, jobURL, nil
}

func sendMessageToSlackChannel(message []slack.Block) error {
	if *dryRun {
		fmt.Println("=== DRY RUN MODE - Message Preview ===")
		messageJSON, err := json.MarshalIndent(message, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal message to JSON: %w", err)
		}
		fmt.Println(string(messageJSON))
		fmt.Println("\n=== Formatted Message ===")
		printFormattedMessage(message)
		return nil
	}

	api := slack.New(token)
	_, _, err := api.PostMessageContext(
		context.Background(),
		channelId,
		slack.MsgOptionBlocks(message...),
	)
	return err
}

func printFormattedMessage(blocks []slack.Block) {
	for _, block := range blocks {
		if section, ok := block.(*slack.SectionBlock); ok {
			if section.Text != nil {
				fmt.Println(section.Text.Text)
			}
		}
	}
}

func generateStatusMessage(buildStatus finished, jobURL, buildId string) []slack.Block {
	buildTime := buildStatus.getBuildTime()
	now := time.Now().UTC()

	// Check if the build is older than 24 hours
	if now.Sub(buildTime) > 24*time.Hour {
		return []slack.Block{
			slack.NewSectionBlock(
				slack.NewTextBlockObject(
					"mrkdwn",
					":warning: *kubernetes-nmstate Future Periodic Job Status*",
					false,
					false,
				),
				nil,
				nil,
			),
			slack.NewSectionBlock(
				slack.NewTextBlockObject(
					"mrkdwn",
					fmt.Sprintf("*Status:* No build in the last 24 hours\n*Last Build:* %s\n*Build Time:* %s",
						buildId,
						buildTime.Format("2006-01-02 15:04:05 UTC")),
					false,
					false,
				),
				nil,
				nil,
			),
		}
	}

	var statusEmoji string
	var statusText string

	if buildStatus.Passed {
		statusEmoji = ":white_check_mark:"
		statusText = "PASSED"
	} else {
		statusEmoji = ":x:"
		statusText = "FAILED"
	}

	return []slack.Block{
		slack.NewSectionBlock(
			slack.NewTextBlockObject(
				"mrkdwn",
				fmt.Sprintf("%s *kubernetes-nmstate Future Periodic Job Status*", statusEmoji),
				false,
				false,
			),
			nil,
			nil,
		),
		slack.NewSectionBlock(
			slack.NewTextBlockObject(
				"mrkdwn",
				fmt.Sprintf("*Status:* %s\n*Build:* <%s|%s>\n*Result:* %s\n*Build Time:* %s\n*Revision:* %s",
					statusText,
					jobURL,
					buildId,
					buildStatus.Result,
					buildTime.Format("2006-01-02 15:04:05 UTC"),
					buildStatus.Revision),
				false,
				false,
			),
			nil,
			nil,
		),
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
