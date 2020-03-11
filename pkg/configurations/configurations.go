package configurations

import (
	"fmt"
	"github.com/thoas/go-funk"
	"os"
	"path/filepath"
	"sync"

	"github.com/gobwas/glob"

	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	"github.com/spf13/viper"
)

const defaultInterfacesFilter = "veth*"

var (
	log   = logf.Log.WithName("configurations")
	mutex = sync.Mutex{}
)

// Struct to match config file
type configParams struct {
	InterfaceFilter      string `mapstructure:"interfaces_filter"`
	InterfacesFilterGlob glob.Glob
}

type config struct {
	configPath   string
	v            *viper.Viper
	configParams configParams
	initialized  bool
}

func newConfig() (*config, error) {
	configPathTemp, isSet := os.LookupEnv("CONFIG_PATH")
	if !isSet {
		return nil, fmt.Errorf("CONFIG_PATH is mandatory")
	}

	configDir, _ := filepath.Split(configPathTemp)
	if !pathExists(configDir) {
		return nil, fmt.Errorf("folder %s doesn't exist, can't start configuration watcher", configDir)
	}

	ext := filepath.Ext(configPathTemp)
	if !funk.Contains(viper.SupportedExts, ext[1:]) {
		return nil, fmt.Errorf("file extension %s is not supported", ext)
	}

	c := config{
		configPath:  configPathTemp,
		initialized: true,
		configParams: configParams{
			InterfaceFilter:      defaultInterfacesFilter,
			InterfacesFilterGlob: glob.MustCompile(defaultInterfacesFilter),
		},
	}

	c.v = viper.New()
	c.v.SetConfigFile(c.configPath)
	c.v.SetTypeByDefaultValue(true)

	return &c, nil
}

var globalConfig config

func CreateGlobalConfig() error {
	mutex.Lock()
	defer mutex.Unlock()
	if globalConfig.initialized {
		return nil
	}
	conf, err := newConfig()
	if err != nil {
		return err
	}
	globalConfig = *conf
	return nil
}

func GetConfigPath() string {
	return globalConfig.configPath
}

func GetCurrentConfig() configParams {
	mutex.Lock()
	defer mutex.Unlock()
	return globalConfig.configParams
}

func GetInterfacesFilterGlob() glob.Glob {
	mutex.Lock()
	defer mutex.Unlock()
	return globalConfig.configParams.InterfacesFilterGlob
}

// Updating relevant values with new config settings
func SetConfig() {
	if globalConfig.initialized == false {
		log.Info("Config is not initialized, can't set values")
		return
	}
	mutex.Lock()
	defer mutex.Unlock()

	if err := globalConfig.v.ReadInConfig(); err != nil { // Find and read the config file
		log.Info("Not able to read configuration, will update default values")
	}

	log.Info("Updating configuration")
	err := globalConfig.v.Unmarshal(&globalConfig.configParams)

	if err != nil {
		log.Error(err, "Failed to unmarshal config file, skipping configuration update")
		return
	}

	globalConfig.configParams.InterfacesFilterGlob = glob.MustCompile(globalConfig.configParams.InterfaceFilter)
	log.Info(fmt.Sprintf("New configuration is %+v", globalConfig.configParams))
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return true
}
