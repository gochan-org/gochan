package main

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	libgeo "github.com/nranchev/go-libGeoIP"
	"golang.org/x/crypto/bcrypt"
)

var (
	nullTime                 time.Time
	errEmptyDurationString   = errors.New("Empty Duration string")
	errInvalidDurationString = errors.New("Invalid Duration string")
	durationRegexp           = regexp.MustCompile(`^((\d+)\s?ye?a?r?s?)?\s?((\d+)\s?mon?t?h?s?)?\s?((\d+)\s?we?e?k?s?)?\s?((\d+)\s?da?y?s?)?\s?((\d+)\s?ho?u?r?s?)?\s?((\d+)\s?mi?n?u?t?e?s?)?\s?((\d+)\s?s?e?c?o?n?d?s?)?$`)
)

const (
	chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 abcdefghijklmnopqrstuvwxyz~!@#$%%^&*()_+{}[]-=:\"\\/?.>,<;:'"
)

func arrToString(arr []string) string {
	var out string
	for i, val := range arr {
		out += val
		if i < len(arr)-1 {
			out += ","
		}
	}
	return out
}

func benchmarkTimer(name string, givenTime time.Time, starting bool) (returnTime time.Time) {
	if starting {
		// starting benchmark test
		println(2, "Starting benchmark \""+name+"\"")
		returnTime = givenTime
	} else {
		// benchmark is finished, print the duration
		// convert nanoseconds to a decimal seconds
		printf(2, "benchmark %s completed in %f seconds\n", name, time.Since(givenTime).Seconds())
		returnTime = time.Now() // we don't really need this, but we have to return something
	}
	return
}

func md5Sum(str string) string {
	hash := md5.New()
	io.WriteString(hash, str)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func sha1Sum(str string) string {
	hash := sha1.New()
	io.WriteString(hash, str)
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func bcryptSum(str string) string {
	digest, err := bcrypt.GenerateFromPassword([]byte(str), 4)
	if err == nil {
		return string(digest)
	}
	return ""
}

func byteByByteReplace(input, from, to string) string {
	if len(from) != len(to) {
		return ""
	}
	for i := 0; i < len(from); i++ {
		input = strings.Replace(input, from[i:i+1], to[i:i+1], -1)
	}
	return input
}

// for easier defer cleaning
func closeHandle(handle io.Closer) {
	if handle != nil && !reflect.ValueOf(handle).IsNil() {
		handle.Close()
	}
}

/*
 * Deletes files in a folder (root) that match a given regular expression.
 * Returns the number of files that were deleted, and any error encountered.
 */
func deleteMatchingFiles(root, match string) (filesDeleted int, err error) {
	files, err := ioutil.ReadDir(root)
	if err != nil {
		return 0, err
	}
	for _, f := range files {
		match, _ := regexp.MatchString(match, f.Name())
		if match {
			os.Remove(filepath.Join(root, f.Name()))
			filesDeleted++
		}
	}
	return filesDeleted, err
}

// getBoardArr performs a query against the database, and returns an array of Boards along with an error value.
// If specified, the string where is added to the query, prefaced by WHERE. An example valid value is where = "id = 1".
func getBoardArr(parameterList map[string]interface{}, extra string) (boards []Board, err error) {
	queryString := "SELECT * FROM " + config.DBprefix + "boards "
	numKeys := len(parameterList)
	var parameterValues []interface{}
	if numKeys > 0 {
		queryString += "WHERE "
	}

	for key, value := range parameterList {
		queryString += fmt.Sprintf("%s = ? AND ", key)
		parameterValues = append(parameterValues, value)
	}

	// Find and remove any trailing instances of "AND "
	if numKeys > 0 {
		queryString = queryString[:len(queryString)-4]
	}

	queryString += fmt.Sprintf(" %s ORDER BY list_order", extra)

	rows, err := querySQL(queryString, parameterValues...)
	defer closeHandle(rows)
	if err != nil {
		handleError(0, "error getting board list: %s", customError(err))
		return
	}

	// For each row in the results from the database, populate a new Board instance,
	// 	then append it to the boards array we are going to return
	for rows.Next() {
		board := new(Board)
		if err = rows.Scan(
			&board.ID,
			&board.ListOrder,
			&board.Dir,
			&board.Type,
			&board.UploadType,
			&board.Title,
			&board.Subtitle,
			&board.Description,
			&board.Section,
			&board.MaxFilesize,
			&board.MaxPages,
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
		); err != nil {
			handleError(0, customError(err))
			return
		}
		boards = append(boards, *board)
	}
	return
}

func getBoardFromID(id int) (*Board, error) {
	board := new(Board)
	err := queryRowSQL("SELECT list_order,dir,type,upload_type,title,subtitle,description,section,"+
		"max_file_size,max_pages,default_style,locked,created_on,anonymous,forced_anon,max_age,"+
		"autosage_after,no_images_after,max_message_length,embeds_allowed,redirect_to_thread,require_file,"+
		"enable_catalog FROM "+config.DBprefix+"boards WHERE id = ?",
		[]interface{}{id},
		[]interface{}{
			&board.ListOrder, &board.Dir, &board.Type, &board.UploadType, &board.Title,
			&board.Subtitle, &board.Description, &board.Section, &board.MaxFilesize,
			&board.MaxPages, &board.DefaultStyle, &board.Locked, &board.CreatedOn,
			&board.Anonymous, &board.ForcedAnon, &board.MaxAge, &board.AutosageAfter,
			&board.NoImagesAfter, &board.MaxMessageLength, &board.EmbedsAllowed,
			&board.RedirectToThread, &board.RequireFile, &board.EnableCatalog,
		},
	)
	board.ID = id
	return board, err
}

// if parameterList is nil, ignore it and treat extra like a whole SQL query
func getPostArr(parameterList map[string]interface{}, extra string) (posts []Post, err error) {
	queryString := "SELECT * FROM " + config.DBprefix + "posts "
	numKeys := len(parameterList)
	var parameterValues []interface{}
	if numKeys > 0 {
		queryString += "WHERE "
	}

	for key, value := range parameterList {
		queryString += fmt.Sprintf("%s = ? AND ", key)
		parameterValues = append(parameterValues, value)
	}

	// Find and remove any trailing instances of "AND "
	if numKeys > 0 {
		queryString = queryString[:len(queryString)-4]
	}

	queryString += " " + extra // " ORDER BY `order`"
	rows, err := querySQL(queryString, parameterValues...)
	defer closeHandle(rows)
	if err != nil {
		handleError(1, customError(err))
		return
	}

	// For each row in the results from the database, populate a new Post instance,
	// then append it to the posts array we are going to return
	for rows.Next() {
		var post Post

		if err = rows.Scan(&post.ID, &post.BoardID, &post.ParentID, &post.Name, &post.Tripcode,
			&post.Email, &post.Subject, &post.MessageHTML, &post.MessageText, &post.Password, &post.Filename,
			&post.FilenameOriginal, &post.FileChecksum, &post.Filesize, &post.ImageW,
			&post.ImageH, &post.ThumbW, &post.ThumbH, &post.IP, &post.Capcode, &post.Timestamp,
			&post.Autosage, &post.DeletedTimestamp, &post.Bumped, &post.Stickied, &post.Locked, &post.Reviewed,
		); err != nil {
			handleError(0, customError(err))
			return
		}
		posts = append(posts, post)
	}
	return
}

// TODO: replace where with a map[string]interface{} like getBoardsArr()
func getSectionArr(where string) (sections []BoardSection, err error) {
	if where == "" {
		where = "1 = 1"
	}
	rows, err := querySQL("SELECT * FROM " + config.DBprefix + "sections WHERE " + where + " ORDER BY list_order")
	defer closeHandle(rows)
	if err != nil {
		handleError(0, err.Error())
		return
	}

	for rows.Next() {
		var section BoardSection
		if err = rows.Scan(&section.ID, &section.ListOrder, &section.Hidden, &section.Name, &section.Abbreviation); err != nil {
			handleError(1, customError(err))
			return
		}
		sections = append(sections, section)
	}
	return
}

func getCountryCode(ip string) (string, error) {
	if config.EnableGeoIP && config.GeoIPDBlocation != "" {
		gi, err := libgeo.Load(config.GeoIPDBlocation)
		if err != nil {
			return "", err
		}
		return gi.GetLocationByIP(ip).CountryCode, nil
	}
	return "", nil
}

func generateSalt() string {
	salt := make([]byte, 3)
	salt[0] = chars[rand.Intn(86)]
	salt[1] = chars[rand.Intn(86)]
	salt[2] = chars[rand.Intn(86)]
	return string(salt)
}

func getFileExtension(filename string) (extension string) {
	if !strings.Contains(filename, ".") {
		extension = ""
	} else {
		extension = filename[strings.LastIndex(filename, ".")+1:]
	}
	return
}

func getFormattedFilesize(size float64) string {
	if size < 1000 {
		return fmt.Sprintf("%dB", int(size))
	} else if size <= 100000 {
		return fmt.Sprintf("%fKB", size/1024)
	} else if size <= 100000000 {
		return fmt.Sprintf("%fMB", size/1024.0/1024.0)
	}
	return fmt.Sprintf("%0.2fGB", size/1024.0/1024.0/1024.0)
}

// returns the filename, line number, and function where getMetaInfo() is called
// stackOffset increases/decreases which item on the stack is referenced.
//	see documentation for runtime.Caller() for more info
func getMetaInfo(stackOffset int) (string, int, string) {
	pc, file, line, _ := runtime.Caller(1 + stackOffset)
	return file, line, runtime.FuncForPC(pc).Name()
}

func customError(err error) string {
	if err != nil {
		file, line, _ := getMetaInfo(2)
		return fmt.Sprintf("[ERROR] %s:%d: %s\n", file, line, err.Error())
	}
	return ""
}

func handleError(verbosity int, format string, a ...interface{}) string {
	out := fmt.Sprintf(format, a...)
	println(verbosity, out)
	errorLog.Print(out)
	return out
}

func humanReadableTime(t time.Time) string {
	return t.Format(config.DateTimeFormat)
}

func getThumbnailPath(thumbType string, img string) string {
	filetype := strings.ToLower(img[strings.LastIndex(img, ".")+1:])
	if filetype == "gif" || filetype == "webm" {
		filetype = "jpg"
	}
	index := strings.LastIndex(img, ".")
	if index < 0 || index > len(img) {
		return ""
	}
	thumbSuffix := "t." + filetype
	if thumbType == "catalog" {
		thumbSuffix = "c." + filetype
	}
	return img[0:index] + thumbSuffix
}

// findResource searches for a file in the given paths and returns the first one it finds
// or a blank string if none of the paths exist
func findResource(paths ...string) string {
	var err error
	for _, filepath := range paths {
		if _, err = os.Stat(filepath); err == nil {
			return filepath
		}
	}
	return ""
}

// paginate returns a 2d array of a specified interface from a 1d array passed in,
// with a specified number of values per array in the 2d array.
// interfaceLength is the number of interfaces per array in the 2d array (e.g, threads per page)
// interf is the array of interfaces to be split up.
func paginate(interfaceLength int, interf []interface{}) [][]interface{} {
	// paginatedInterfaces = the finished interface array
	// numArrays = the current number of arrays (before remainder overflow)
	// interfacesRemaining = if greater than 0, these are the remaining interfaces
	// 	that will be added to the super-interface

	var paginatedInterfaces [][]interface{}
	numArrays := len(interf) / interfaceLength
	interfacesRemaining := len(interf) % interfaceLength
	currentInterface := 0
	for l := 0; l < numArrays; l++ {
		paginatedInterfaces = append(paginatedInterfaces,
			interf[currentInterface:currentInterface+interfaceLength])
		currentInterface += interfaceLength
	}
	if interfacesRemaining > 0 {
		paginatedInterfaces = append(paginatedInterfaces, interf[len(interf)-interfacesRemaining:])
	}
	return paginatedInterfaces
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
	allBoards = nil
	allSections = nil

	allBoardsArr, _ := getBoardArr(nil, "")
	for _, b := range allBoardsArr {
		allBoards = append(allBoards, b)
	}

	allSectionsArr, _ := getSectionArr("")
	allSections = append(allSections, allSectionsArr...)
}

func searchStrings(item string, arr []string, permissive bool) int {
	for i, str := range arr {
		if item == str {
			return i
		}
	}
	return -1
}

func bToI(b bool) int {
	if b {
		return 1
	}
	return 0
}

func bToA(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// Checks the validity of the Akismet API key given in the config file.
func checkAkismetAPIKey(key string) error {
	if key == "" {
		return fmt.Errorf("Blank key given, Akismet spam checking won't be used.")
	}
	resp, err := http.PostForm("https://rest.akismet.com/1.1/verify-key", url.Values{"key": {key}, "blog": {"http://" + config.SiteDomain}})
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(body) == "invalid" {
		// This should disable the Akismet checks if the API key is not valid.
		errmsg := "Akismet API key is invalid, Akismet spam protection will be disabled."
		return fmt.Errorf(errmsg)
	}
	return nil
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
			handleError(1, err.Error())
			return "other_failure"
		}
		req.Header.Set("User-Agent", "gochan/1.0 | Akismet/0.1")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		defer func() {
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
		}()
		if err != nil {
			handleError(1, err.Error())
			return "other_failure"
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			handleError(1, err.Error())
			return "other_failure"
		}
		errorLog.Print("Response from Akismet: " + string(body))

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

func marshalJSON(tag string, data interface{}, indent bool) (string, error) {
	var jsonBytes []byte
	var err error

	if tag != "" {
		data = map[string]interface{}{
			tag: data,
		}
	}
	if indent {
		jsonBytes, err = json.MarshalIndent(data, "", "	")
	} else {
		jsonBytes, err = json.Marshal(data)
	}

	if err != nil {
		jsonBytes, _ = json.Marshal(map[string]string{"error": err.Error()})
	}
	return string(jsonBytes), err
}

func limitArraySize(arr []string, maxSize int) []string {
	if maxSize > len(arr)-1 || maxSize < 0 {
		return arr
	}
	return arr[:maxSize]
}

func numReplies(boardid, threadid int) int {
	var num int

	if err := queryRowSQL(
		"SELECT COUNT(*) FROM "+config.DBprefix+"posts WHERE boardid = ? AND parentid = ?",
		[]interface{}{boardid, threadid}, []interface{}{&num}); err != nil {
		return 0
	}
	return num
}

func ipMatch(newIP, existingIP string) bool {
	if newIP == existingIP {
		// both are single IPs and are the same
		return true
	}
	wildcardIndex := strings.Index(existingIP, "*")
	if wildcardIndex < 0 {
		// single (or invalid) and they don't match
		return false
	}
	ipRegexStr := existingIP[0:wildcardIndex]
	ipRegexStr = strings.Replace(ipRegexStr, ".", "\\.", -1) + ".*"
	ipRegex, err := regexp.Compile(ipRegexStr)
	if err != nil {
		// this shouldn't happen unless you enter an invalid IP in the db
		return false
	}
	return ipRegex.MatchString(newIP)
}

// based on TinyBoard's parse_time function
func parseDurationString(str string) (time.Duration, error) {
	if str == "" {
		return 0, errEmptyDurationString
	}

	matches := durationRegexp.FindAllStringSubmatch(str, -1)
	if len(matches) == 0 {
		return 0, errInvalidDurationString
	}

	var expire int
	if matches[0][2] != "" {
		years, _ := strconv.Atoi(matches[0][2])
		expire += years * 60 * 60 * 24 * 365
	}
	if matches[0][4] != "" {
		months, _ := strconv.Atoi(matches[0][4])
		expire += months * 60 * 60 * 24 * 30
	}
	if matches[0][6] != "" {
		weeks, _ := strconv.Atoi(matches[0][6])
		expire += weeks * 60 * 60 * 24 * 7
	}
	if matches[0][8] != "" {
		days, _ := strconv.Atoi(matches[0][8])
		expire += days * 60 * 60 * 24
	}
	if matches[0][10] != "" {
		hours, _ := strconv.Atoi(matches[0][10])
		expire += hours * 60 * 60
	}
	if matches[0][12] != "" {
		minutes, _ := strconv.Atoi(matches[0][12])
		expire += minutes * 60
	}
	if matches[0][14] != "" {
		seconds, _ := strconv.Atoi(matches[0][14])
		expire += seconds
	}
	return time.ParseDuration(strconv.Itoa(expire) + "s")
}
