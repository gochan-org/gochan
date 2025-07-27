package posting

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
)

var (
	ErrNoCaptchaToken     = errors.New("missing required CAPTCHA")
	ErrUnsupportedCaptcha = errors.New("unsupported captcha type set in configuration (currently only hcaptcha is supported)")
	validCaptchaTypes     = []string{"hcaptcha"}
)

type CaptchaResult struct {
	Hostname  string    `json:"hostname"`
	Credit    bool      `json:"credit"`
	Success   bool      `json:"success"`
	Timestamp time.Time `json:"challenge_ts"`
}

func InitCaptcha() error {
	captchaCfg := config.GetSiteConfig().Captcha
	if captchaCfg == nil {
		return nil
	}

	if !slices.Contains(validCaptchaTypes, captchaCfg.Type) {
		return ErrUnsupportedCaptcha
	}
	return nil
}

// submitCaptchaResponse parses the incoming captcha form values, submits them, and returns the results
func submitCaptchaResponse(request *http.Request) (bool, error) {
	captchaCfg := config.GetSiteConfig().Captcha
	if captchaCfg == nil {
		return true, nil // captcha isn't required, skip the test
	}
	threadid, _ := strconv.Atoi(request.PostFormValue("threadid"))
	if captchaCfg.OnlyNeededForThreads && threadid > 0 {
		return true, nil
	}

	var token string
	switch captchaCfg.Type {
	case "hcaptcha":
		token = request.PostFormValue("h-captcha-response")
	default:

	}

	if token == "" {
		return false, ErrNoCaptchaToken
	}
	params := url.Values{
		"secret":   []string{captchaCfg.AccountSecret},
		"response": []string{token},
	}
	resp, err := http.PostForm("https://hcaptcha.com/siteverify", params)
	if err != nil {
		return false, err
	}
	var vals CaptchaResult
	if err = json.NewDecoder(resp.Body).Decode(&vals); err != nil {
		return false, err
	}
	return vals.Success, resp.Body.Close()
}

// ServeCaptcha handles requests to /captcha if the captcha is properly configured
func ServeCaptcha(writer http.ResponseWriter, request *http.Request) {
	captchaCfg := config.GetSiteConfig().Captcha
	if request.Method == "GET" && request.FormValue("needcaptcha") != "" {
		fmt.Fprint(writer, captchaCfg != nil)
		return
	}
	errEv := gcutil.LogError(nil).
		Str("IP", gcutil.GetRealIP(request))
	defer func() {
		errEv.Discard()
	}()
	wantsJSON := serverutil.IsRequestingJSON(request)
	if captchaCfg == nil {
		server.ServeError(writer, "This site is not set up to require a CAPTCHA test", wantsJSON, nil)
		return
	}
	if request.Method == "POST" {
		result, err := submitCaptchaResponse(request)
		if err != nil {
			errEv.Err(err).Caller().Msg("Error submitting CAPTCHA")
			server.ServeError(writer, "Error checking CAPTCHA results: "+err.Error(), wantsJSON, nil)
			return
		}
		gcutil.LogInfo().
			Bool("result", result).
			Str("IP", gcutil.GetRealIP(request)).
			Msg("Got CAPTCHA result")
	}
	var buf bytes.Buffer
	err := serverutil.MinifyTemplate(gctemplates.Captcha, map[string]any{
		"boardConfig": config.GetBoardConfig(""),
		"boards":      gcsql.AllBoards,
		"siteKey":     captchaCfg.SiteKey,
	}, &buf, "text/html")
	if err != nil {
		errEv.Err(err).Caller().Msg("Unable to build CAPTCHA template")
		server.ServeError(writer, "Error serving CAPTCHA: "+err.Error(), wantsJSON, nil)
	}
	buf.WriteTo(writer)
}
