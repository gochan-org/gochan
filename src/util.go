package main

import (
	"crypto/md5"
	"crypto/sha1"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	null_time, _ = time.Parse("2006-01-02 15:04:05", "0000-00-00 00:00:00")
)

const (
	chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 abcdefghijklmnopqrstuvwxyz~!@#$%%^&*()_+{}[]-=:\"\\/?.>,<;:'"
)

func benchmarkTimer(name string, given_time time.Time, starting bool) time.Time {
	if starting {
		// starting benchmark test
		println(2, "Starting benchmark \""+name+"\"")
		return given_time
	} else {
		// benchmark is finished, print the duration
		// convert nanoseconds to a decimal seconds
		printf(2, "benchmark %s completed in %f seconds", name, time.Since(given_time).Seconds())
		return time.Now() // we don't really need this, but we have to return something
	}
}

func md5_sum(str string) string {
	hash := md5.New()
	io.WriteString(hash, str)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func sha1_sum(str string) string {
	hash := sha1.New()
	io.WriteString(hash, str)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func bcrypt_sum(str string) string {
	hash := ""
	digest, err := bcrypt.GenerateFromPassword([]byte(str), 4)
	if err == nil {
		hash = string(digest)
	}
	return hash
}

func byteByByteReplace(input, from, to string) string {
	if len(from) != len(to) {
		return ""
	}
	for i := 0; i < len(from); i += 1 {
		input = strings.Replace(input, from[i:i+1], to[i:i+1], -1)
	}
	return input
}

// Deletes files in a folder (root) that match a given regular expression.
// Returns the number of files that were deleted, and any error encountered.
func deleteMatchingFiles(root, match string) (files_deleted int, err error) {
	files, err := ioutil.ReadDir(root)
	if err != nil {
		return 0, err
	}
	for _, f := range files {
		match, _ := regexp.MatchString(match, f.Name())
		if match {
			os.Remove(filepath.Join(root, f.Name()))
			files_deleted++
		}
	}
	return files_deleted, err
}

// getBoardArr performs a query against the database, and returns an array of BoardsTables along with an error value.
// If specified, the string where is added to the query, prefaced by WHERE. An example valid value is where = "id = 1".
func getBoardArr(where string) (boards []BoardsTable, err error) {
	if where == "" {
		where = "1"
	}
	rows, err := db.Query("SELECT * FROM `" + config.DBprefix + "boards` WHERE " + where + " ORDER BY `order`;")
	if err != nil {
		error_log.Print(err.Error())
		return
	}

	// For each row in the results from the database, populate a new BoardsTable instance,
	// 	then append it to the boards array we are going to return
	for rows.Next() {
		board := new(BoardsTable)
		err = rows.Scan(
			&board.ID,
			&board.Order,
			&board.Dir,
			&board.Type,
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

func getPostArr(sql string) (posts []interface{}, err error) {
	rows, err := db.Query(sql)
	if err != nil {
		error_log.Print(err.Error())
		return
	}
	for rows.Next() {
		var post PostTable
		err = rows.Scan(&post.ID, &post.BoardID, &post.ParentID, &post.Name, &post.Tripcode,
			&post.Email, &post.Subject, &post.MessageHTML, &post.MessageText, &post.Password, &post.Filename,
			&post.FilenameOriginal, &post.FileChecksum, &post.Filesize, &post.ImageW,
			&post.ImageH, &post.ThumbW, &post.ThumbH, &post.IP, &post.Tag, &post.Timestamp,
			&post.Autosage, &post.PosterAuthority, &post.DeletedTimestamp, &post.Bumped,
			&post.Stickied, &post.Locked, &post.Reviewed, &post.Sillytag)
		if err != nil {
			error_log.Print("util.go:getPostArr() ERROR: " + err.Error())
			return
		}
		posts = append(posts, post)
	}
	return
}

func getSectionArr(where string) (sections []interface{}, err error) {
	if where == "" {
		where = "1"
	}
	rows, err := db.Query("SELECT * FROM `" + config.DBprefix + "sections` WHERE " + where + " ORDER BY `order`;")
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
		//} else if strings.Index(filename, "/") > -1 {
	} else {
		return filename[strings.LastIndex(filename, ".")+1:]
	}
}

func getFormattedFilesize(size float32) string {
	if size < 1000 {
		return fmt.Sprintf("%fB", size)
	} else if size <= 100000 {
		return fmt.Sprintf("%fKB", size/1024)
	} else if size <= 100000000 {
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

// paginate returns a 2d array of a specified interface from a 1d array passed in,
//	with a specified number of values per array in the 2d array.
// interface_length is the number of interfaces per array in the 2d array (e.g, threads per page)
// interf is the array of interfaces to be split up.
func paginate(interface_length int, interf []interface{}) [][]interface{} {
	// paginated_interfaces = the finished interface array
	// num_arrays = the current number of arrays (before remainder overflow)
	// interfaces_remaining = if greater than 0, these are the remaining interfaces
	// 		that will be added to the super-interface

	var paginated_interfaces [][]interface{}
	num_arrays := len(interf) / interface_length
	interfaces_remaining := len(interf) % interface_length
	//paginated_interfaces = append(paginated_interfaces, interf)
	current_interface := 0
	for l := 0; l < num_arrays; l++ {
		paginated_interfaces = append(paginated_interfaces,
			interf[current_interface:current_interface+interface_length])
		current_interface += interface_length
	}
	if interfaces_remaining > 0 {
		paginated_interfaces = append(paginated_interfaces, interf[len(interf)-interfaces_remaining:])
	}
	return paginated_interfaces
}

func printf(v int, format string, a ...interface{}) {
	if config.Verbosity >= v {
		fmt.Printf(format, a...)
	}
}

func println(v int, a ...interface{}) {
	if config.Verbosity >= v {
		fmt.Println(a...)
	}
}

func resetBoardSectionArrays() {
	// run when the board list needs to be changed (board/section is added, deleted, etc)
	all_boards = nil
	all_sections = nil

	all_boards_a, _ := getBoardArr("")
	for _, b := range all_boards_a {
		all_boards = append(all_boards, b)
	}
	all_sections_a, _ := getSectionArr("")
	for _, b := range all_sections_a {
		all_sections = append(all_sections, b)
	}
}

func searchStrings(item string, arr []string, permissive bool) int {
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

// Checks the validity of the Akismet API key given in the config file.
func checkAkismetAPIKey() {
	resp, err := http.PostForm("https://rest.akismet.com/1.1/verify-key", url.Values{"key": {config.AkismetAPIKey}, "blog": {"http://" + config.SiteDomain}})
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		error_log.Print(err.Error())
	}
	if string(body) == "invalid" {
		// This should disable the Akismet checks if the API key is not valid.
		error_log.Print("Akismet API key is invalid, Akismet spam protection will be disabled.")
		config.AkismetAPIKey = ""
	}
}

// Checks a given post for spam with Akismet. Only checks if Akismet API key is set.
func checkPostForSpam(userIP string, userAgent string, referrer string,
	author string, email string, postContent string) string {
	if config.AkismetAPIKey != "" {
		client := &http.Client{}
		data := url.Values{"blog": {"http://" + config.SiteDomain}, "user_ip": {userIP}, "user_agent": {userAgent}, "referrer": {referrer},
			"comment_type": {"forum-post"}, "comment_author": {author}, "comment_author_email": {email},
			"comment_content": {postContent}}

		req, err := http.NewRequest("POST", "https://"+config.AkismetAPIKey+".rest.akismet.com/1.1/comment-check",
			strings.NewReader(data.Encode()))
		if err != nil {
			error_log.Print(err.Error())
			return "other_failure"
		}
		req.Header.Set("User-Agent", "gochan/1.0 | Akismet/0.1")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		if err != nil {
			error_log.Print(err.Error())
			return "other_failure"
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			error_log.Print(err.Error())
			return "other_failure"
		}
		error_log.Print("Response from Akismet: " + string(body))

		if string(body) == "true" {
			if proTip, ok := resp.Header["X-akismet-pro-tip"]; ok && proTip[0] == "discard" {
				return "discard"
			}
			return "spam"
		} else if string(body) == "invalid" {
			return "invalid"
		} else if string(body) == "false" {
			return "ham"
		}
	}
	return "other_failure"
}

func makePostJSON(post PostTable, anonymous string) (post_obj PostJSON) {
	var filename string
	var fileext string
	var orig_filename string

	// Separate out the extension from the filenames
	if post.Filename != "deleted" && post.Filename != "" {
		ext_start := strings.LastIndex(post.Filename, ".")
		fileext = post.Filename[ext_start:]

		orig_ext_start := strings.LastIndex(post.FilenameOriginal, fileext)
		orig_filename = post.FilenameOriginal[:orig_ext_start]
		filename = post.Filename[:ext_start]
	}

	post_obj = PostJSON { ID: post.ID, ParentID: post.ParentID, Subject: post.Subject, Message: post.MessageHTML,
		Name: post.Name, Timestamp: post.Timestamp.Unix(), Bumped: post.Bumped.Unix(),
		ThumbWidth: post.ThumbW, ThumbHeight: post.ThumbH, ImageWidth: post.ImageW, ImageHeight: post.ImageH,
		FileSize: post.Filesize, OrigFilename: orig_filename, Extension: fileext, Filename: filename, FileChecksum: post.FileChecksum}

	// Handle Anonymous
	if post.Name == "" {
		post_obj.Name = anonymous
	}

	// If we have a Tripcode, prepend a !
	if post.Tripcode != "" {
		post_obj.Tripcode = "!" + post.Tripcode
	}
	return
}
