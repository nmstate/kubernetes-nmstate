package nmstatectl

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"

	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	nmstate "github.com/nmstate/kubernetes-nmstate/pkg/apis/nmstate/shared"
)

var (
	log       = logf.Log.WithName("nmstatectl")
	statePipe = "/tmp/state.pipe"
)

const nmstateCommand = "nmstatectl"

func nmstatectlWithInput(arguments []string, input string) (string, error) {
	cmd := exec.Command(nmstateCommand, arguments...)
	var stdout, stderr bytes.Buffer
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if input != "" {
		stdin, err := cmd.StdinPipe()
		if err != nil {
			return "", fmt.Errorf("failed to create pipe for writing into %s: %v", nmstateCommand, err)
		}
		go func() {
			defer stdin.Close()
			_, err = io.WriteString(stdin, input)
			if err != nil {
				fmt.Printf("failed to write input into stdin: %v\n", err)
			}
		}()

	}
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to execute %s %s: '%v' '%s' '%s'", nmstateCommand, strings.Join(arguments, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String(), nil

}

func nmstatectl(arguments []string) (string, error) {
	return nmstatectlWithInput(arguments, "")
}

func Show() (string, error) {
	fifo, err := os.OpenFile(statePipe, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		return "", errors.Wrap(err, "failed opening named pipe to read nmstatectl show")
	}
	defer fifo.Close()
	var buff bytes.Buffer
	io.Copy(&buff, fifo)
	return buff.String(), nil
}

func Set(desiredState nmstate.State, timeout time.Duration) (string, error) {
	var setDoneCh = make(chan struct{})
	go setUnavailableUp(setDoneCh)
	defer close(setDoneCh)

	setOutput, err := nmstatectlWithInput([]string{"set", "--no-commit", "--timeout", strconv.Itoa(int(timeout.Seconds()))}, string(desiredState.Raw))
	return setOutput, err
}

func Commit() (string, error) {
	return nmstatectl([]string{"commit"})
}

func Rollback() error {
	_, err := nmstatectl([]string{"rollback"})
	if err != nil {
		return errors.Wrapf(err, "failed calling nmstatectl rollback")
	}
	return nil
}

func StartMonitor() error {
	os.Remove(statePipe)
	err := syscall.Mkfifo(statePipe, 0666)
	if err != nil {
		return errors.Wrap(err, "failed creating the named pipe to talk with nmstatectl")
	}

	cmd := exec.Command("nmstatectl-monitor", statePipe)
	if err := cmd.Start(); err != nil {
		return errors.Wrap(err, "failed starting nmstatectl-monitor command")
	}
	return nil
}
