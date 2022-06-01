package app

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gookit/color"
	"github.com/gookit/config/v2"
	"github.com/gookit/config/v2/dotnev"
	"github.com/gookit/config/v2/toml"
	appconfig "github.com/omnibuildplatform/omni-repository/common/config"
	"os"
	"path/filepath"
)

var (
	AppConfig appconfig.Config
)

func Bootstrap(configDir, tag, commitID, releaseAt string) {
	//Load configIn(app.TimeZone)
	loadConfig(configDir)
	//Initialize environment
	initAppEnv()
	//init app
	Info = ApplicationInfo{
		Tag:       tag,
		CommitID:  commitID,
		ReleaseAt: releaseAt,
	}
	initAppInfo()
	//init logger
	initLogger()
	color.Info.Printf(
		"\n============ Bootstrap (EnvName: %s, Debug: %v) ============\n",
		EnvName, Debug,
	)
}

func initAppInfo() {
	//update App info
	Name = config.String("name", DefaultAppName)

}

func loadConfig(configDir string) {
	files, err := getConfigFiles(configDir)
	if err != nil {
		color.Error.Printf("failed to load config files in folder %s %v\n", configDir, err)
		os.Exit(1)
	}
	cfg := config.Default()
	config.AddDriver(toml.Driver)
	err = cfg.LoadFiles(files...)
	if err != nil {
		color.Error.Println("failed to load config files %v", err)
		os.Exit(1)
	}
	err = config.BindStruct("", &AppConfig)
	if err != nil {
		color.Error.Println("config file mismatched with current config object %v", err)
		os.Exit(1)
	}
}

func getConfigFiles(configDir string) ([]string, error) {
	var files = make([]string, 0)
	err := filepath.Walk(configDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		//valid files
		//1. app.toml
		//2. dev|test|prod.app.toml
		if info.Name() == BaseConfigFile || info.Name() == fmt.Sprintf("%s.%s", EnvName, BaseConfigFile) {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return files, err
	}
	return files, nil
}

func initAppEnv() {
	//load env from .env file
	err := dotnev.LoadExists(".", ".env")
	if err != nil {
		color.Error.Println(err.Error())
	}

	Hostname, _ = os.Hostname()
	if env := os.Getenv("APP_ENV"); env != "" {
		EnvName = env
	}
	if EnvName == EnvDev || EnvName == EnvTest {
		gin.SetMode(gin.DebugMode)
		Debug = true
	} else {
		gin.SetMode(gin.ReleaseMode)
		Debug = false
	}
}
