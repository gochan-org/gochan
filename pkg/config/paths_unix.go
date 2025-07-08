//go:build unix && !darwin

package config

var (
	StandardConfigSearchPaths = []string{"gochan.json", "/usr/local/etc/gochan/gochan.json", "/etc/gochan/gochan.json"}
)
