package config

type (
	Config struct {
		Name         string          `mapstructure:"name"`
		ServerConfig ServerConfig    `mapstructure:"server"`
		RepoManager  RepoManager     `mapstructure:"repoManager"`
		LogConfig    LogConfig       `mapstructure:"log"`
		Store        PersistentStore `mapstructure:"persistentStore"`
		WorkManager  WorkManager     `mapstructure:"workManager"`
		MQ           MQ              `mapstructure:"mq"`
	}

	ServerConfig struct {
		PublicHttpPort   int    `mapstructure:"publicHttpPort"`
		InternalHttpPort int    `mapstructure:"internalHttpPort"`
		DataFolder       string `mapstructure:"dataFolder"`
	}

	LogConfig struct {
		LogFile string `mapstructure:"logFile"`
		ErrFile string `mapstructure:"errFile"`
	}

	RepoManager struct {
		UploadToken string `mapstructure:"uploadToken"`
		CallBackUrl string `mapstructure:"callBackUrl"`
	}

	WorkManager struct {
		Worker       int `mapstructure:"worker"`
		SyncInterval int `mapstructure:"syncInterval"`
	}

	PersistentStore struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		DBName   string `mapstructure:"dbname"`
	}

	MQ struct {
		KafkaBrokers string `mapstructure:"kafka_brokers"`
	}
)
