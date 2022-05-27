package config

type (
	Config struct {
		Name         string          `mapstructure:"name"`
		ServerConfig ServerConfig    `mapstructure:"server"`
		RepoManager  RepoManager     `mapstructure:"repoManager"`
		LogConfig    LogConfig       `mapstructure:"log"`
		Store        PersistentStore `mapstructure:"persistentStore"`
	}

	ServerConfig struct {
		HttpPort int `mapstructure:"httpPort"`
	}

	LogConfig struct {
		LogFile string `mapstructure:"logFile"`
		ErrFile string `mapstructure:"errFile"`
	}

	RepoManager struct {
		DataFolder  string `mapstructure:"dataFolder"`
		UploadToken string `mapstructure:"uploadToken"`
		CallBackUrl string `mapstructure:"callBackUrl"`
	}

	PersistentStore struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		DBName   string `mapstructure:"dbname"`
	}
)
