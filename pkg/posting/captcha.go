package posting

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
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

func InitCaptcha() {
	var typeIsValid bool
	captchaCfg := config.GetSiteConfig().Captcha
	if !captchaCfg.UseCaptcha() {
		return
	}
	for _, vType := range validCaptchaTypes {
		if captchaCfg.Type == vType {
			typeIsValid = true
		}
	}
	if !typeIsValid {
		fmt.Printf("Unrecognized Captcha.Type value in configuration: %q, valid values: %v\n",
			captchaCfg.Type, validCaptchaTypes)
		gcutil.LogFatal().
			Str("captchaType", captchaCfg.Type).
			Msg("Unsupported captcha type set in configuration")
	}
}

// submitCaptchaResponse parses the incoming captcha form values, submits them, and returns the results
func submitCaptchaResponse(request *http.Request) (bool, error) {
	captchaCfg := config.GetSiteConfig().Captcha
	if !captchaCfg.UseCaptcha() {
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
		fmt.Fprint(writer, captchaCfg.UseCaptcha())
		return
	}
	errEv := gcutil.LogError(nil).
		Str("IP", gcutil.GetRealIP(request))
	defer func() {
		errEv.Discard()
	}()
	wantsJSON := serverutil.IsRequestingJSON(request)
	if !captchaCfg.UseCaptcha() {
		server.ServeError(writer, "This site is not set up to require a CAPTCHA test", wantsJSON, nil)
		return
	}
	if request.Method == "POST" {
		result, err := submitCaptchaResponse(request)
		if err != nil {
			errEv.Err(err).Caller().Send()
			server.ServeError(writer, "Error checking CAPTCHA results: "+err.Error(), wantsJSON, nil)
			return
		}
		fmt.Println("Success:", result)
	}
	err := serverutil.MinifyTemplate(gctemplates.Captcha, map[string]interface{}{
		"boardConfig": config.GetBoardConfig(""),
		"boards":      gcsql.AllBoards,
		"siteKey":     captchaCfg.SiteKey,
	}, writer, "text/html")
	if err != nil {
		errEv.Err(err).Caller().Send()
		server.ServeError(writer, "Error serving CAPTCHA: "+err.Error(), wantsJSON, nil)
	}
}
