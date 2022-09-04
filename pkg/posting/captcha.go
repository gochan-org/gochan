package posting

import (
	"fmt"
	"image/color"
	"net/http"
	"strconv"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/serverutil"
	"github.com/mojocn/base64Captcha"
)

var (
	captchaString *base64Captcha.DriverString
	driver        *base64Captcha.DriverString
)

type captchaJSON struct {
	CaptchaID     string `json:"id"`
	Base64String  string `json:"image"`
	Result        string `json:"-"`
	TempPostIndex string `json:"-"`
	EmailCmd      string `json:"-"`
}

// InitCaptcha prepares the captcha driver for use
func InitCaptcha() {
	boardConfig := config.GetBoardConfig("")
	if !boardConfig.UseCaptcha {
		return
	}
	driver = base64Captcha.NewDriverString(
		boardConfig.CaptchaHeight, boardConfig.CaptchaWidth, int(0), int(0), int(6),
		"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ",
		&color.RGBA{0, 0, 0, 0}, nil, nil).ConvertFonts()
}

// ServeCaptcha handles requests to /captcha if UseCaptcha is enabled in gochan.json
func ServeCaptcha(writer http.ResponseWriter, request *http.Request) {
	boardConfig := config.GetBoardConfig("")
	if !boardConfig.UseCaptcha {
		return
	}
	var err error
	if err = request.ParseForm(); err != nil {
		gcutil.LogError(err).Msg("Failed parsing request form")
		serverutil.ServeErrorPage(writer, "Error parsing request form: "+err.Error())
		return
	}

	tempPostIndexStr := request.FormValue("temppostindex")
	var tempPostIndex int
	if tempPostIndex, err = strconv.Atoi(tempPostIndexStr); err != nil {
		tempPostIndexStr = "-1"
		tempPostIndex = 0
	}
	emailCommand := request.FormValue("emailcmd")

	id, b64 := getCaptchaImage()
	captchaStruct := captchaJSON{id, b64, "", tempPostIndexStr, emailCommand}
	useJSON := request.FormValue("json") == "1"
	if useJSON {
		writer.Header().Add("Content-Type", "application/json")

		str, _ := gcutil.MarshalJSON(captchaStruct, false)
		serverutil.MinifyWriter(writer, []byte(str), "application/json")
		return
	}
	if request.FormValue("reload") == "Reload" {
		request.Form.Del("reload")
		request.Form.Add("didreload", "1")
		ServeCaptcha(writer, request)
		return
	}
	writer.Header().Add("Content-Type", "text/html")
	captchaID := request.FormValue("captchaid")
	captchaAnswer := request.FormValue("captchaanswer")
	if captchaID != "" && request.FormValue("didreload") != "1" {
		goodAnswer := base64Captcha.DefaultMemStore.Verify(captchaID, captchaAnswer, true)
		if goodAnswer {
			if tempPostIndex > -1 && tempPostIndex < len(gcsql.TempPosts) {
				// came from a /post redirect, insert the specified temporary post
				// and redirect to the thread

				gcsql.InsertPost(&gcsql.TempPosts[tempPostIndex], emailCommand == "noko")
				building.BuildBoards(false, gcsql.TempPosts[tempPostIndex].BoardID)
				building.BuildFrontPage()

				url := gcsql.TempPosts[tempPostIndex].GetURL(false)

				// move the end Post to the current index and remove the old end Post. We don't
				// really care about order as long as tempPost validation doesn't get jumbled up
				gcsql.TempPosts[tempPostIndex] = gcsql.TempPosts[len(gcsql.TempPosts)-1]
				gcsql.TempPosts = gcsql.TempPosts[:len(gcsql.TempPosts)-1]
				http.Redirect(writer, request, url, http.StatusFound)
				return
			}
		} else {
			captchaStruct.Result = "Incorrect CAPTCHA"
		}
	}
	if err = serverutil.MinifyTemplate(gctemplates.Captcha, captchaStruct, writer, "text/html"); err != nil {
		gcutil.LogError(err).
			Str("template", "captcha").Send()
		fmt.Fprint(writer, "Error executing captcha template: ", err.Error())
	}
}

func getCaptchaImage() (captchaID, chaptchaB64 string) {
	boardConfig := config.GetBoardConfig("")
	if !boardConfig.UseCaptcha {
		return
	}
	captcha := base64Captcha.NewCaptcha(driver, base64Captcha.DefaultMemStore)
	captchaID, chaptchaB64, _ = captcha.Generate()
	return
}
