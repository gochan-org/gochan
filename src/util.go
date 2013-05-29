package main

import (
	"crypto/md5"
	"crypto/sha1"
	"code.google.com/p/go.crypto/bcrypt"
	"io"
	"math/rand"
	"net/http"
	"fmt"
	"strconv"
	"time"
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
	digest,err := bcrypt.GenerateFromPassword([]byte(str), 4)
	if err == nil {
		//hash = fmt.Sprintf("%x",digest)
		hash = string(digest)
	}
	return hash
}

func getBoardArr() (boards []interface{}) {
  	results,err := db.Start("SELECT * FROM `"+config.DBprefix+"boards` ORDER BY `order`;")
	if err != nil {
		error_log.Write(err.Error())
		return 
	}
	rows,err := results.GetRows()
	if err != nil {
		error_log.Write(err.Error())
		return
	}

	for _,row := range rows {
		var board BoardsTable
		board.IName = "board"
		board.ID,_ = strconv.Atoi(string(row[0].([]byte)))
		board.Order,_ = strconv.Atoi(string(row[1].([]byte)))
		board.Dir = string(row[2].([]byte))
		board.Type,_ = strconv.Atoi(string(row[3].([]byte)))
		board.FirstPost,_ = strconv.Atoi(string(row[4].([]byte)))
		board.UploadType,_ = strconv.Atoi(string(row[5].([]byte)))
		board.Title = string(row[6].([]byte))
		board.Subtitle = string(row[7].([]byte))
		board.Description = string(row[8].([]byte))
		board.Section,_ = strconv.Atoi(string(row[9].([]byte)))
		board.MaxImageSize,_ = strconv.Atoi(string(row[10].([]byte)))
		board.MaxPages,_ = strconv.Atoi(string(row[11].([]byte)))
		board.Locale = string(row[12].([]byte))
		board.DefaultStyle = string(row[13].([]byte))
		board.Locked = (string(row[14].([]byte)) == "1")
		board.CreatedOn = string(row[15].([]byte))
		board.Anonymous = string(row[16].([]byte))
		board.ForcedAnon = string(row[17].([]byte))
		board.MaxAge,_ = strconv.Atoi(string(row[18].([]byte)))
		board.MarkPage,_ = strconv.Atoi(string(row[19].([]byte)))
		board.AutosageAfter,_ = strconv.Atoi(string(row[20].([]byte)))
		board.NoImagesAfter,_ = strconv.Atoi(string(row[21].([]byte)))
		board.MaxMessageLength,_ = strconv.Atoi(string(row[22].([]byte)))
		board.EmbedsAllowed = string(row[23].([]byte))
		board.RedirectToThread = (string(row[24].([]byte)) == "1")
		board.ShowId = (string(row[25].([]byte)) == "1")
		board.CompactList = (string(row[26].([]byte)) == "1")
		board.EnableNofile = (string(row[27].([]byte)) == "1")
		board.EnableCatalog = (string(row[28].([]byte)) == "1")
		boards = append(boards, board)
	}
	return
}

func getSectionArr() (sections []interface{}) {
	results,err := db.Start("SELECT * FROM `"+config.DBprefix+"sections` ORDER BY `order`;")
	if err != nil {
		error_log.Write(err.Error())
		return
	}
	rows,err := results.GetRows()
	if err != nil {
		error_log.Write(err.Error())
		return
	}

	for _,row := range rows {
		var section BoardSectionsTable
		section.IName = "section"
		section.ID,_ = strconv.Atoi(string(row[0].([]byte)))
		section.Order,_ = strconv.Atoi(string(row[1].([]byte)))
		section.Hidden = (string(row[2].([]byte)) == "1")
		section.Name = string(row[3].([]byte))
		section.Abbreviation = string(row[3].([]byte))
		sections = append(sections, section)
	}
	return
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

func getSQLDateTime() string {
	now := time.Now()
	return now.Format("2006-01-02 15:04:05")
}

func getSpecificSQLDateTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
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

