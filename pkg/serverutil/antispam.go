package serverutil

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
)

// CheckAkismetAPIKey checks the validity of the Akismet API key given in the config file.
func CheckAkismetAPIKey(key string) error {
	if key == "" {
		return errors.New("blank key given, Akismet spam checking won't be used")
	}
	resp, err := http.PostForm("https://rest.akismet.com/1.1/verify-key", url.Values{"key": {key}, "blog": {"http://" + config.Config.SiteDomain}})
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
		errmsg := "Akismet API key is invalid, Akismet spam protection will be disabled."
		gclog.Print(gclog.LErrorLog, errmsg)
		return errors.New(errmsg)
	}
	return nil
}

// CheckPostForSpam checks a given post for spam with Akismet. Only checks if Akismet API key is set.
func CheckPostForSpam(userIP, userAgent, referrer, author, email, postContent string) string {
	if config.Config.AkismetAPIKey != "" {
		client := &http.Client{}
		data := url.Values{"blog": {"http://" + config.Config.SiteDomain}, "user_ip": {userIP}, "user_agent": {userAgent}, "referrer": {referrer},
			"comment_type": {"forum-post"}, "comment_author": {author}, "comment_author_email": {email},
			"comment_content": {postContent}}

		req, err := http.NewRequest("POST", "https://"+config.Config.AkismetAPIKey+".rest.akismet.com/1.1/comment-check",
			strings.NewReader(data.Encode()))
		if err != nil {
			gclog.Print(gclog.LErrorLog, err.Error())
			return "other_failure"
		}
		req.Header.Set("User-Agent", "gochan/1.0 | Akismet/0.1")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		if err != nil {
			gclog.Print(gclog.LErrorLog, err.Error())
			return "other_failure"
		}
		if resp.Body != nil {
			resp.Body.Close()
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			gclog.Print(gclog.LErrorLog, err.Error())
			return "other_failure"
		}
		gclog.Print(gclog.LErrorLog, "Response from Akismet: ", string(body))

		if string(body) == "true" {
			if proTip, ok := resp.Header["X-akismet-pro-tip"]; ok && proTip[0] == "discard" {
				return "discard"
			}
			return "spam"
		} else if string(body) == "invalid" {
			return "invalid"
		} else if string(body) == "false" {
			return "ham"
		}
	}
	return "other_failure"
}

// ValidReferer checks to make sure that the incoming request is from the same domain (or if debug mode is enabled)
func ValidReferer(request *http.Request) bool {
	if config.Config.DebugMode {
		return true
	}
	rURL, err := url.ParseRequestURI(request.Referer())
	if err != nil {
		gclog.Println(gclog.LAccessLog|gclog.LErrorLog, "Error parsing referer URL:", err.Error())
		return false
	}

	return strings.Index(rURL.Path, config.Config.SiteWebfolder) == 0
}
