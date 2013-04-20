package main

import (
	"crypto/md5"
	"crypto/sha1"
	"code.google.com/p/go.crypto/bcrypt"
	"io"
	"math/rand"
	"net/http"
	"fmt"
	"unsafe"
)
// #cgo LDFLAGS: -lcrypt
// #define _GNU_SOURCE
// #include <crypt.h>
// #include <stdlib.h>
import "C"

var crypt_data = C.struct_crypt_data{}

const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 abcdefghijklmnopqrstuvwxyz~!@#$%^&*()_+{}[]-=:\"\\/?.>,<;:'"


func crypt(key, salt string) string {
	ckey := C.CString(key)
	csalt := C.CString(salt)
	out := C.GoString(C.crypt_r(ckey,csalt,&crypt_data))
	C.free(unsafe.Pointer(ckey))
	C.free(unsafe.Pointer(csalt))
	return out
}

func md5_sum(str string) string {
	hash := md5.New()
	io.WriteString(hash, str)
	digest := fmt.Sprintf("%x",hash.Sum(nil))
	return digest
}

func sha1_sum(str string) string {
	hash := sha1.New()
	io.WriteString(hash,str)
	digest := fmt.Sprintf("%x",hash.Sum(nil))
	return digest
}

func bcrypt_sum(str string) string {
	hash := ""
	digest,err := bcrypt.GenerateFromPassword([]byte(str), 10)
	if err == nil {
		hash = fmt.Sprintf("%x",digest)	
	}
	return hash
}

func getCookie(name string) *http.Cookie {
	num_cookies := len(cookies)
	for c := 0; c < num_cookies; c += 1 {
		if cookies[c].Name == name {
			return cookies[c]
		}
	}
	return nil
}

func generateSalt() string {
	salt := make([]byte, 3)
	salt[0] = chars[rand.Intn(86)]
	salt[1] = chars[rand.Intn(86)]
	salt[2] = chars[rand.Intn(86)]
	return string(salt)
}

func getFormattedFilesize(size float32) string {
	if(size < 1000) {
		return fmt.Sprintf("%fB", size)
	} else if(size <= 100000) {
		//size = size * 0.2
		return fmt.Sprintf("%fKB", size/1024)
	} else if(size <= 100000000) {
		//size = size * 0.2
		return fmt.Sprintf("%fMB", size/1024/1024)
	}
	return fmt.Sprintf("%0.2fGB", size/1024/1024/1024)
}

func searchStrings(item string,arr []string,permissive bool) int {
	var length = len(arr)
	for i := 0; i < length; i++ {
		if item == arr[i] {
			return i
		}
	}
	return -1
}

func Btoi(b bool) int {
	if b == true { return 1 }
	return 0
}

