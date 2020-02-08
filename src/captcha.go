package main

import (
	"fmt"
	"image/color"
	"net/http"
	"strconv"

	//"github.com/mojocn/base64Captcha"
	"gopkg.in/mojocn/base64Captcha.v1"
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

func initCaptcha() {
	if !config.UseCaptcha {
		return
	}
	driver = base64Captcha.NewDriverString(
		config.CaptchaHeight, config.CaptchaWidth, 0, 0, 6,
		"0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ",
		&color.RGBA{0, 0, 0, 0}, nil).ConvertFonts()
}

func serveCaptcha(writer http.ResponseWriter, request *http.Request) {
	if !config.UseCaptcha {
		return
	}
	var err error
	if err = request.ParseForm(); err != nil {
		serveErrorPage(writer, err.Error())
		errorLog.Println(customError(err))
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
		str, _ := marshalJSON("", captchaStruct, false)
		minifyWriter(writer, []byte(str), "application/json")
		return
	}
	if request.FormValue("reload") == "Reload" {
		request.Form.Del("reload")
		request.Form.Add("didreload", "1")
		serveCaptcha(writer, request)
		return
	}
	writer.Header().Add("Content-Type", "text/html")
	captchaID := request.FormValue("captchaid")
	captchaAnswer := request.FormValue("captchaanswer")
	if captchaID != "" && request.FormValue("didreload") != "1" {
		goodAnswer := base64Captcha.DefaultMemStore.Verify(captchaID, captchaAnswer, true)
		if goodAnswer {
			if tempPostIndex > -1 && tempPostIndex < len(tempPosts) {
				// came from a /post redirect, insert the specified temporary post
				// and redirect to the thread
				insertPost(&tempPosts[tempPostIndex], emailCommand == "noko")
				buildBoards(tempPosts[tempPostIndex].BoardID)
				buildFrontPage()
				url := tempPosts[tempPostIndex].GetURL(false)

				// move the end Post to the current index and remove the old end Post. We don't
				// really care about order as long as tempPost validation doesn't get jumbled up
				tempPosts[tempPostIndex] = tempPosts[len(tempPosts)-1]
				tempPosts = tempPosts[:len(tempPosts)-1]
				http.Redirect(writer, request, url, http.StatusFound)
				return
			}
		} else {
			captchaStruct.Result = "Incorrect CAPTCHA"
		}
	}
	if err = minifyTemplate(captchaTmpl, captchaStruct, writer, "text/html"); err != nil {
		handleError(0, customError(err))
		fmt.Fprintf(writer, "Error executing captcha template")
	}
}

func getCaptchaImage() (captchaID string, chaptchaB64 string) {
	if !config.UseCaptcha {
		return
	}
	captcha := base64Captcha.NewCaptcha(driver, base64Captcha.DefaultMemStore)
	captchaID, chaptchaB64, _ = captcha.Generate()
	return
}
