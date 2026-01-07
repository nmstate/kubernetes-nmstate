# kubernetes-nmstate Latest Periodic Job Reporter

This is a Slack bot that reports the status of the `periodic-knmstate-e2e-handler-k8s-latest` periodic job.

## Overview

The reporter runs daily via GitHub Actions and posts a message to a configured Slack channel with the status of the latest periodic job run. It monitors the periodic job that runs the latest version of nmstate against kubernetes-nmstate.

## Configuration

The following GitHub secret needs to be configured in the repository:

- `NMSTATE_SLACK_WEBHOOK_URL`: The Slack incoming webhook URL for posting messages

### Command Line Flags

- `--notify-on`: Events to notify on (can be specified multiple times). Valid values: `success`, `failure`, `stale`. Default: all events
- `--fake`: Generate a fake report for testing. Valid values: `success`, `failure`, `stale`
- `--dry-run`: Print the message without sending it to Slack

## How it Works

1. The GitHub Action runs daily at 5:30 AM UTC (configurable via cron schedule)
2. The reporter fetches the latest build information from the Prow storage bucket
3. It checks the build status and timestamp
4. Based on the `--notify-on` configuration, a Slack message may be posted for:
   - **Success**: Build passed - posts a success message with build details
   - **Failure**: Build failed - posts a failure message with build details
   - **Stale**: No build in the last 24 hours - posts a warning message

By default, the GitHub Action is configured to only notify on failures and stale builds to reduce noise.

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
cd automation/nmstate-latest-reporter

# Set required environment variable
export NMSTATE_SLACK_WEBHOOK_URL="your-webhook-url"

# Build and run
go build -v .
./nmstate-latest-reporter

# Only notify on failures and stale builds (skip success)
./nmstate-latest-reporter --notify-on=failure --notify-on=stale

# Only notify on failures
./nmstate-latest-reporter --notify-on=failure
```

### Testing with Fake Reports

You can generate fake reports for testing without fetching real Prow data:

```bash
# Generate a fake success report
./nmstate-latest-reporter --fake=success

# Generate a fake failure report
./nmstate-latest-reporter --fake=failure

# Generate a fake stale report
./nmstate-latest-reporter --fake=stale

# Test with dry-run (doesn't send to Slack)
./nmstate-latest-reporter --fake=success --dry-run

# Test notification filtering with fake data
./nmstate-latest-reporter --fake=success --notify-on=failure --dry-run
```

This is useful for:
- Testing the Slack integration without waiting for real builds
- Verifying message formatting and appearance
- Testing notification workflows

## Prow Job Details

The reporter monitors the following periodic job:
- **Job Name**: `periodic-knmstate-e2e-handler-k8s-latest`
- **Schedule**: Daily at 2:00 AM UTC
- **Purpose**: Tests kubernetes-nmstate with the latest version of nmstate
- **Storage**: `gs://kubevirt-prow/logs/periodic-knmstate-e2e-handler-k8s-latest`
