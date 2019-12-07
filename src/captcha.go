package main

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/mojocn/base64Captcha"
)

var (
	charCaptchaCfg base64Captcha.ConfigCharacter
)

type captchaJSON struct {
	CaptchaID     string `json:"id"`
	Base64String  string `json:"image"`
	Result        string `json:"-"`
	TempPostIndex string `json:"-"`
	EmailCmd      string `json:"-"`
}

func initCaptcha() {
	charCaptchaCfg = base64Captcha.ConfigCharacter{
		Height:             config.CaptchaHeight, // originally 60
		Width:              config.CaptchaWidth,  // originally 240
		Mode:               base64Captcha.CaptchaModeNumberAlphabet,
		ComplexOfNoiseText: base64Captcha.CaptchaComplexLower,
		ComplexOfNoiseDot:  base64Captcha.CaptchaComplexLower,
		IsUseSimpleFont:    true,
		IsShowHollowLine:   false,
		IsShowNoiseDot:     true,
		IsShowNoiseText:    false,
		IsShowSlimeLine:    true,
		IsShowSineLine:     false,
		CaptchaLen:         8,
	}
}

func serveCaptcha(writer http.ResponseWriter, request *http.Request) {
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
		goodAnswer := base64Captcha.VerifyCaptcha(captchaID, captchaAnswer)
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
	var captchaInstance base64Captcha.CaptchaInterface
	captchaID, captchaInstance = base64Captcha.GenerateCaptcha("", charCaptchaCfg)
	chaptchaB64 = base64Captcha.CaptchaWriteToBase64Encoding(captchaInstance)
	return
}
