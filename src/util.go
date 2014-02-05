package main

import (
	"crypto/md5"
	"crypto/sha1"
	"code.google.com/p/go.crypto/bcrypt"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unsafe"
)
// #cgo LDFLAGS: -lcrypt
// #define _GNU_SOURCE
// #include <crypt.h>
// #include <stdlib.h>
import "C"

var (
	crypt_data = C.struct_crypt_data{}
)

const (
	chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 abcdefghijklmnopqrstuvwxyz~!@#$%%^&*()_+{}[]-=:\"\\/?.>,<;:'"
)


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
	return fmt.Sprintf("%x",hash.Sum(nil))
}

func sha1_sum(str string) string {
	hash := sha1.New()
	io.WriteString(hash,str)
	return fmt.Sprintf("%x",hash.Sum(nil))
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

func byteByByteReplace(input,from, to string) string {
	if len(from) != len(to) {
		return ""
	}
	for i := 0; i < len(from); i += 1 {
		input = strings.Replace(input, from[i:i+1], to[i:i+1], -1)
	}
	return input
}

func walkfolder(path string, f os.FileInfo, err error) error {
	if err != nil {
		return err
	}
	if !f.IsDir() {
		return os.Remove(path)
	} else {
		return nil
	}
}

func deleteFolderContents(root string) error {
	err := filepath.Walk(root, walkfolder)
	return err
}

func getBoardArr(where string) (boards []BoardsTable) {
	if where == "" {
		where = "1"
	}
	rows,err := db.Query("SELECT * FROM `"+config.DBprefix+"boards` WHERE "+where+" ORDER BY `order`;")
	if err != nil {
		error_log.Print(err.Error())
		return
	}

	for rows.Next() {
		board := new(BoardsTable)
		err = rows.Scan(
			&board.ID,
			&board.Order,
			&board.Dir,
			&board.Type,
			&board.FirstPost,
			&board.UploadType,
			&board.Title,
			&board.Subtitle,
			&board.Description,
			&board.Section,
			&board.MaxImageSize,
			&board.MaxPages,
			&board.Locale, 
			&board.DefaultStyle, 
			&board.Locked, 
			&board.CreatedOn, 
			&board.Anonymous, 
			&board.ForcedAnon, 
			&board.MaxAge, 
			&board.AutosageAfter, 
			&board.NoImagesAfter,
			&board.MaxMessageLength,
			&board.EmbedsAllowed,
			&board.RedirectToThread,
			&board.ShowId,
			&board.RequireFile,
			&board.EnableCatalog,
		)
		board.IName = "board"
		if err != nil {
			error_log.Print(err.Error())
			fmt.Println(err.Error())
			return
		} else {
			boards = append(boards, *board)
		}
	}
	return
}

func getPostArr(sql string) (posts []interface{},err error) {
	rows,err := db.Query(sql)
	if err != nil {
		error_log.Print(err.Error())
		return
	}
	for rows.Next() {
		var post PostTable
		err = rows.Scan(&post.ID, &post.BoardID, &post.ParentID, &post.Name, &post.Tripcode, &post.Email, &post.Subject, &post.Message, &post.Password, &post.Filename, &post.FilenameOriginal, &post.FileChecksum, &post.Filesize, &post.ImageW, &post.ImageH, &post.ThumbW, &post.ThumbH, &post.IP, &post.Tag, &post.Timestamp, &post.Autosage, &post.PosterAuthority, &post.DeletedTimestamp, &post.Bumped, &post.Stickied, &post.Locked, &post.Reviewed, &post.Sillytag)
		if err != nil {
			error_log.Print(err.Error())
			return
		}
		posts = append(posts, post)
	}
	return posts,err
}

func getSectionArr(where string) (sections []interface{}) {
	if where == "" {
		where = "1"
	}
	rows,err := db.Query("SELECT * FROM `"+config.DBprefix+"sections` WHERE "+where+" ORDER BY `order`;")
	if err != nil {
		error_log.Print(err.Error())
		return
	}

	for rows.Next() {
		section := new(BoardSectionsTable)
		section.IName = "section"

		err = rows.Scan(&section.ID, &section.Order, &section.Hidden, &section.Name, &section.Abbreviation)
		if err != nil {
			error_log.Print(err.Error())
			return
		}
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

func getFileExtension(filename string) string {
	if strings.Index(filename, ".") == -1 {
		return ""
	} else if strings.Index(filename, "/") > -1 {
		return filename[strings.LastIndex(filename,".")+1:]
	}
	return ""
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
	return now.Format(mysql_datetime_format)
}

func getSpecificSQLDateTime(t time.Time) string {
	return t.Format(mysql_datetime_format)
}

func humanReadableTime(t time.Time) string {
	return t.Format(config.DateTimeFormat)
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
	if b == true {
		return 1
	}
	return 0
}

func Btoa(b bool) string {
	if b == true {
		return "1"
	}
	return "0"
}