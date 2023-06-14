package main

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/gochan-org/gochan/pkg/events"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

var (
	basicUrlRE       = regexp.MustCompile(`https?://\S+\.(\S+)`)
	recognizedTLDs   = []string{"com", "net", "org", "edu", "gov", "us", "uk"}
	errUntrustedURLs = errors.New("post contains one or more untrusted links")
)

func hasUntrustedTLD(msg string) bool {
	urls := basicUrlRE.FindAllStringSubmatch(msg, -1)
	for _, match := range urls {
		tld := match[1]
		trusted := false
		for _, recognized := range recognizedTLDs {
			if tld == recognized {
				trusted = true
				break
			}
		}
		if !trusted {
			return true
		}
	}
	return false
}

func isNewPoster(post *gcsql.Post) (bool, error) {
	var ipCount int
	err := gcsql.QueryRowSQL(`SELECT COUNT(*) FROM DBPREFIXposts WHERE ip = ?`, []any{post.IP}, []any{&ipCount})
	return ipCount == 0, err
}

func InitPlugin() error {
	events.RegisterEvent([]string{"message-pre-format"}, func(trigger string, args ...interface{}) error {
		if len(args) == 0 {
			return nil
		}

		post, ok := args[0].(*gcsql.Post)
		if !ok {
			// argument isn't actually a post
			return fmt.Errorf(events.InvalidArgumentErrorStr, trigger)
		}

		newPoster, err := isNewPoster(post)
		if err != nil {
			return err
		}

		if newPoster && hasUntrustedTLD(post.MessageRaw) {
			return errUntrustedURLs
		}

		return nil
	})

	return nil
}
