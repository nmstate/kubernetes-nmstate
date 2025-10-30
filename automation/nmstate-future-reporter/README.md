# kubernetes-nmstate Future Periodic Job Reporter

This is a Slack bot that reports the status of the `periodic-knmstate-e2e-handler-k8s-future` periodic job.

## Overview

The reporter runs daily via GitHub Actions and posts a message to a configured Slack channel with the status of the latest periodic job run. It monitors the periodic job that runs the future version of nmstate against kubernetes-nmstate.

## Configuration

The following GitHub secret needs to be configured in the repository:

- `NMSTATE_SLACK_WEBHOOK_URL`: The Slack incoming webhook URL for posting messages

## How it Works

1. The GitHub Action runs daily at 5:30 AM UTC (configurable via cron schedule)
2. The reporter fetches the latest build information from the Prow storage bucket
3. It checks the build status and timestamp
4. A Slack message is posted with one of the following scenarios:
   - **Success**: Build passed - posts a success message with build details
   - **Failure**: Build failed - posts a failure message with build details
   - **No Recent Build**: No build in the last 24 hours - posts a warning message

## Message Format

The Slack messages include:
- Build status (PASSED/FAILED)
- Build ID with link to the Prow job
- Build result
- Build timestamp
- Git revision

## Manual Execution

The workflow can be manually triggered using the GitHub Actions `workflow_dispatch` event.

## Local Development

To run the reporter locally:

```bash
cd automation/nmstate-future-reporter

# Set required environment variable
export NMSTATE_SLACK_WEBHOOK_URL="your-webhook-url"

# Build and run
go build -v .
./nmstate-future-reporter
```

### Testing with Fake Reports

You can generate fake reports for testing without fetching real Prow data:

```bash
# Generate a fake success report
./nmstate-future-reporter -fake success

# Generate a fake failure report
./nmstate-future-reporter -fake failure
```

This is useful for:
- Testing the Slack integration without waiting for real builds
- Verifying message formatting and appearance
- Testing notification workflows

## Prow Job Details

The reporter monitors the following periodic job:
- **Job Name**: `periodic-knmstate-e2e-handler-k8s-future`
- **Schedule**: Daily at 2:00 AM UTC
- **Purpose**: Tests kubernetes-nmstate with the future version of nmstate
- **Storage**: `gs://kubevirt-prow/logs/periodic-knmstate-e2e-handler-k8s-future`
