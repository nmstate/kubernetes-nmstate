package configurations

import (
	"fmt"
	"path/filepath"

	"gopkg.in/fsnotify.v1"
)

// This function reads current config file if it is exists.
// If file exists it will update config values with values from it.
// If not it will update config with default values
// And after that it starts configuration watcher go routine
func Start(configFile string, setConfigFunc func()) error {
	configDir, _ := filepath.Split(configFile)
	if !pathExists(configDir) {
		return fmt.Errorf("folder %s doesn't exist, can't start configuration watcher", configDir)
	}
	setConfigFunc()
	watchConfigFile(configFile, setConfigFunc)
	return nil
}

// Adapted from viper WatchConfig to match nmstate configmap watch needs
// The main changes is that there is no need to
// preexist config file before starting the watch
// and it will not exit on file deletion
func watchConfigFile(configPath string, setConfigFunc func()) {
	configFile := filepath.Clean(configPath)
	configDir, _ := filepath.Split(configFile)
	realConfigFile, _ := filepath.EvalSymlinks(configPath)

	newWatcher, err := fsnotify.NewWatcher()

	if err != nil {
		log.Error(err, "Failed to start fsnotify watcher")
		return
	}
	newWatcher.Add(configDir)

	go func() {
		defer newWatcher.Close()
		for {
			select {
			case event, ok := <-newWatcher.Events:
				if !ok { // 'Events' channel is closed
					return
				}
				currentConfigFile, _ := filepath.EvalSymlinks(configPath)
				// we only care about the config file with the following cases:
				// 1 - if the config file was modified or created
				// 2 - if the real path to the config file changed (eg: k8s ConfigMap replacement)
				const writeOrCreateMask = fsnotify.Write | fsnotify.Create

				isModifiedOrCreated := filepath.Clean(event.Name) == configFile && event.Op&writeOrCreateMask != 0
				isrealConfigFileChanged := currentConfigFile != "" && currentConfigFile != realConfigFile

				if isModifiedOrCreated || isrealConfigFileChanged {
					realConfigFile = currentConfigFile
					setConfigFunc()
				}
			case err, ok := <-newWatcher.Errors:
				if ok { // 'Errors' channel is not closed
					log.Error(err, "newWatcher error\n")
				}
				return
			}
		}
	}()
}
