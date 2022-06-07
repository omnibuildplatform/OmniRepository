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
	}

	WorkManager struct {
		SyncInterval int     `mapstructure:"syncInterval"`
		Threads      int     `mapstructure:"threads"`
		Workers      Workers `mapstructure:"workers"`
	}

	PersistentStore struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		DBName   string `mapstructure:"dbname"`
	}

	Workers struct {
		ImagePusher   ImagePusher   `mapstructure:"imagerPusher"`
		ImageVerifier ImageVerifier `mapstructure:"imageVerifier"`
		ImagePuller   ImagePuller   `mapstructure:"imagerPuller"`
	}

	ImagePuller struct {
		MaxRetry       int `mapstructure:"maxRetry"`
		MaxConcurrency int `mapstructure:"maxConcurrency"`
	}

	ImageVerifier struct {
	}

	ImagePusher struct {
		Endpoint string `mapstructure:"endpoint"`
		AK       string `mapstructure:"ak"`
		SK       string `mapstructure:"sk"`
		Bucket   string `mapstructure:"bucket"`
		PartSize int64  `mapstructure:"partSize"`
	}

	MQ struct {
		KafkaBrokers string `mapstructure:"kafka_brokers"`
	}
)
