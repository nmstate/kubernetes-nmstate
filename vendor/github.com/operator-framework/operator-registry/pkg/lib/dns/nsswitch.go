package dns

import (
	"io/ioutil"
	"os"
	"runtime"
)

var (
	GOOS             = runtime.GOOS
	NsswitchContents = []byte("hosts: files dns")
	NsswitchFilename = "/etc/nsswitch.conf"
)

func EnsureNsswitch() error {
	// only linux supports nsswitch
	if GOOS != "linux" {
		return nil
	}

	// if the file already exists, don't overwrite it
	_, err := os.Stat(NsswitchFilename)
	if !os.IsNotExist(err) {
		return nil
	}

	return ioutil.WriteFile(NsswitchFilename, NsswitchContents, 0644)
}
