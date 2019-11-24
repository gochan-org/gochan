package main

import (
	"fmt"
	"testing"
)

func TestGetCaptchaImage(t *testing.T) {
	initCaptcha()
	captchaID, captchaB64 := getCaptchaImage()
	fmt.Println("captchaID:", captchaID, "\ncaptchaB64:", captchaB64)
}
