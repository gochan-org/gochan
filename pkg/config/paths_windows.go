package config

const (
	ConfigNotFoundInPathsMessage = "Unable to load configuration, run gochan-install to generate a new configuration file, or copy gochan.example.json to the current directory and rename it to gochan.json"
)

var (
	StandardConfigSearchPaths []string = []string{"gochan.json"}
)
