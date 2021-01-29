package config

import "os"

// copied from gcutil to avoid import loop
func findResource(paths ...string) string {
	var err error
	for _, filepath := range paths {
		if _, err = os.Stat(filepath); err == nil {
			return filepath
		}
	}
	return ""
}
