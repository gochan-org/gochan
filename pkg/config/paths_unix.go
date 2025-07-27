//go:build unix && !darwin

package config

const (
	ConfigNotFoundInPathsMessage = "Unable to load configuration, run gochan-install to generate a new configuration file, or copy gochan.example.json to one of the search paths and rename it to gochan.json"
)

var (
	StandardConfigSearchPaths = []string{"gochan.json", "/usr/local/etc/gochan/gochan.json", "/etc/gochan/gochan.json"}
)
