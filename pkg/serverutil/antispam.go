package serverutil

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

var (
	ErrBlankAkismetKey   = errors.New("blank Akismet key")
	ErrInvalidAkismetKey = errors.New("invalid Akismet key")
)

// CheckAkismetAPIKey checks the validity of the Akismet API key given in the config file.
func CheckAkismetAPIKey(key string) error {
	if key == "" {
		return ErrBlankAkismetKey
	}
	resp, err := http.PostForm("https://rest.akismet.com/1.1/verify-key", url.Values{"key": {key}, "blog": {"http://" + config.GetSystemCriticalConfig().SiteDomain}})
	if err != nil {
		return err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(body) == "invalid" {
		// This should disable the Akismet checks if the API key is not valid.
		return ErrInvalidAkismetKey
	}
	return nil
}

// CheckPostForSpam checks a given post for spam with Akismet. Only checks if Akismet API key is set.
func CheckPostForSpam(userIP, userAgent, referrer, author, email, postContent string) string {
	systemCritical := config.GetSystemCriticalConfig()
	siteCfg := config.GetSiteConfig()
	if siteCfg.AkismetAPIKey != "" {
		client := &http.Client{}
		data := url.Values{"blog": {"http://" + systemCritical.SiteDomain}, "user_ip": {userIP}, "user_agent": {userAgent}, "referrer": {referrer},
			"comment_type": {"forum-post"}, "comment_author": {author}, "comment_author_email": {email},
			"comment_content": {postContent}}

		req, err := http.NewRequest("POST", "https://"+siteCfg.AkismetAPIKey+".rest.akismet.com/1.1/comment-check",
			strings.NewReader(data.Encode()))
		if err != nil {
			gcutil.Logger().Err(err).
				Str("subject", "akismet").Send()
			return "other_failure"
		}
		req.Header.Set("User-Agent", "gochan/1.0 | Akismet/0.1")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		if err != nil {
			gcutil.Logger().Err(err).
				Str("subject", "akismet").Send()
			return "other_failure"
		}
		if resp.Body != nil {
			resp.Body.Close()
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			gcutil.Logger().Err(err).
				Str("subject", "akismet").Send()
			return "other_failure"
		}
		bodyStr := string(body)
		if config.GetDebugMode() {
			gcutil.Logger().Info().
				Str("subject", "akismet").
				Str("reponse", bodyStr)
		}

		if bodyStr == "true" {
			if proTip, ok := resp.Header["X-akismet-pro-tip"]; ok && proTip[0] == "discard" {
				return "discard"
			}
			return "spam"
		} else if bodyStr == "invalid" {
			return "invalid"
		} else if bodyStr == "false" {
			return "ham"
		}
	}
	return "other_failure"
}

// ValidReferer checks to make sure that the incoming request is from the same domain (or if debug mode is enabled)
func ValidReferer(request *http.Request) bool {
	if config.GetDebugMode() {
		return true
	}
	referer := request.Referer()
	rURL, err := url.ParseRequestURI(referer)
	if err != nil {
		gcutil.Logger().Err(err).
			Str("referer", referer).
			Msg("Error parsing referer URL")
		return false
	}
	return strings.Index(rURL.Path, config.GetSystemCriticalConfig().WebRoot) == 0
}
