package app

type ApplicationInfo struct {
	Tag       string `json:"tag" description:"get tag name"`
	CommitID  string `json:"commitID" description:"git commit ID."`
	ReleaseAt string `json:"releaseAt" description:"build date"`
}

const (
	EnvProd = "prod"
	EnvTest = "test"
	EnvDev  = "dev"
)

const (
	BaseConfigFile = "app.toml"
	DefaultAppName = "omni-repository"
)

var (
	// App name
	Name string
	//Debug mode
	Debug bool
	//Current host name
	Hostname string
	//App port listen to
	//Env name
	EnvName = EnvDev
	//App git info
	Info ApplicationInfo
)
