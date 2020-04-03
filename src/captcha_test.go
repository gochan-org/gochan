package main

import (
	"fmt"
	"testing"
)

func TestGetCaptchaImage(t *testing.T) {
	config.UseCaptcha = true
	config.CaptchaWidth = 240
	config.CaptchaHeight = 80
	initCaptcha()
	captchaID, captchaB64 := getCaptchaImage()
	fmt.Printf("captchaID: %s\ncaptchaB64: %s\n", captchaID, captchaB64)
}
