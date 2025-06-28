//go:build !darwin && !windows

package main

import (
	"slices"

	"github.com/gochan-org/gochan/pkg/config"
)

var (
	cfgPaths = slices.DeleteFunc(config.StandardConfigSearchPaths, func(s string) bool {
		return s == "/opt/homebrew/etc/gochan/gochan.json"
	}) // Exclude Homebrew path on non-macOS systems

)

func init() {
	slices.Reverse(cfgPaths) // /etc/gochan/gochan.json should be first on *nix systems
}
