package networkmanager

import (
	"fmt"
	"os"
)

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed", err)
		os.Exit(1)
	}
}
