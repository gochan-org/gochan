package main

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"database/sql"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/nranchev/go-libGeoIP"
	"golang.org/x/crypto/bcrypt"
)

var (
	nullTime, _ = time.Parse("2006-01-02 15:04:05", "0000-00-00 00:00:00")
)

const (
	chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 abcdefghijklmnopqrstuvwxyz~!@#$%%^&*()_+{}[]-=:\"\\/?.>,<;:'"
)

func benchmarkTimer(name string, givenTime time.Time, starting bool) (returnTime time.Time) {
	if starting {
		// starting benchmark test
		println(2, "Starting benchmark \""+name+"\"")
		returnTime = givenTime
	} else {
		// benchmark is finished, print the duration
		// convert nanoseconds to a decimal seconds
		printf(2, "benchmark %s completed in %f seconds", name, time.Since(givenTime).Seconds())
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
func closeFile(file *os.File) {
	if file != nil {
		_ = file.Close()
	}
}

func closeRows(rows *sql.Rows) {
	if rows != nil {
		_ = rows.Close()
	}
}

func closeStatement(stmt *sql.Stmt) {
	if stmt != nil {
		_ = stmt.Close()
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

// escapeString and escapeQuotes copied from github.com/ziutek/mymysql/native/codecs.go
func escapeString(txt string) string {
	var (
		esc string
		buf bytes.Buffer
	)
	last := 0
	for ii, bb := range txt {
		switch bb {
		case 0:
			esc = `\0`
		case '\n':
			esc = `\n`
		case '\r':
			esc = `\r`
		case '\\':
			esc = `\\`
		case '\'':
			esc = `\'`
		case '"':
			esc = `\"`
		case '\032':
			esc = `\Z`
		default:
			continue
		}
		io.WriteString(&buf, txt[last:ii])
		io.WriteString(&buf, esc)
		last = ii + 1
	}
	io.WriteString(&buf, txt[last:])
	return buf.String()
}

func escapeQuotes(txt string) string {
	var buf bytes.Buffer
	last := 0
	for ii, bb := range txt {
		if bb == '\'' {
			io.WriteString(&buf, txt[last:ii])
			io.WriteString(&buf, `''`)
			last = ii + 1
		}
	}
	io.WriteString(&buf, txt[last:])
	return buf.String()
}

// getBoardArr performs a query against the database, and returns an array of BoardsTables along with an error value.
// If specified, the string where is added to the query, prefaced by WHERE. An example valid value is where = "id = 1".
//func getBoardArr(where string) (boards []BoardsTable, err error) {
func getBoardArr(parameterList map[string]interface{}, extra string) (boards []BoardsTable, err error) {
	queryString := "SELECT * FROM `" + config.DBprefix + "boards` "
	numKeys := len(parameterList)
	var parameterValues []interface{}
	if numKeys > 0 {
		queryString += "WHERE "
	}

	for key, value := range parameterList {
		queryString += fmt.Sprintf("`%s` = ? AND ", key)
		parameterValues = append(parameterValues, value)
	}

	// Find and remove any trailing instances of "AND "
	if numKeys > 0 {
		queryString = queryString[:len(queryString)-4]
	}

	queryString += fmt.Sprintf(" %s ORDER BY `order`", extra)
	printf(2, "queryString@getBoardArr: %s\n", queryString)

	rows, err := querySQL(queryString, parameterValues...)
	defer closeRows(rows)
	if err != nil {
		handleError(0, "error getting board list: %s", customError(err))
		return
	}

	// For each row in the results from the database, populate a new BoardsTable instance,
	// 	then append it to the boards array we are going to return
	for rows.Next() {
		board := new(BoardsTable)
		board.IName = "board"
		if err = rows.Scan(
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
		); err != nil {
			handleError(0, customError(err))
			return
		}
		boards = append(boards, *board)
	}
	return
}

func getBoardFromID(id int) (*BoardsTable, error) {
	board := new(BoardsTable)
	err := queryRowSQL(
		"SELECT `order`,`dir`,`type`,`upload_type`,`title`,`subtitle`,`description`,`section`,"+
			"`max_image_size`,`max_pages`,`locale`,`default_style`,`locked`,`created_on`,`anonymous`,`forced_anon`,`max_age`,"+
			"`autosage_after`,`no_images_after`,`max_message_length`,`embeds_allowed`,`redirect_to_thread`,`require_file`,"+
			"`enable_catalog` FROM `"+config.DBprefix+"boards` WHERE `id` = ?",
		[]interface{}{id},
		[]interface{}{
			&board.Order, &board.Dir, &board.Type, &board.UploadType, &board.Title,
			&board.Subtitle, &board.Description, &board.Section, &board.MaxImageSize,
			&board.MaxPages, &board.Locale, &board.DefaultStyle, &board.Locked, &board.CreatedOn,
			&board.Anonymous, &board.ForcedAnon, &board.MaxAge, &board.AutosageAfter,
			&board.NoImagesAfter, &board.MaxMessageLength, &board.EmbedsAllowed,
			&board.RedirectToThread, &board.RequireFile, &board.EnableCatalog,
		},
	)

	board.ID = id
	return board, err
}

// if parameterList is nil, ignore it and treat extra like a whole SQL query
func getPostArr(parameterList map[string]interface{}, extra string) (posts []PostTable, err error) {
	queryString := "SELECT * FROM `" + config.DBprefix + "posts` "
	numKeys := len(parameterList)
	var parameterValues []interface{}
	if numKeys > 0 {
		queryString += "WHERE "
	}

	for key, value := range parameterList {
		queryString += fmt.Sprintf("`%s` = ? AND ", key)
		parameterValues = append(parameterValues, value)
	}

	// Find and remove any trailing instances of "AND "
	if numKeys > 0 {
		queryString = queryString[:len(queryString)-4]
	}

	queryString += " " + extra // " ORDER BY `order`"
	printf(2, "queryString@getPostArr queryString: %s\n", queryString)

	rows, err := querySQL(queryString, parameterValues...)
	defer closeRows(rows)
	if err != nil {
		handleError(1, customError(err))
		return
	}

	// For each row in the results from the database, populate a new PostTable instance,
	// 	then append it to the posts array we are going to return
	for rows.Next() {
		var post PostTable
		post.IName = "post"
		if err = rows.Scan(&post.ID, &post.BoardID, &post.ParentID, &post.Name, &post.Tripcode,
			&post.Email, &post.Subject, &post.MessageHTML, &post.MessageText, &post.Password, &post.Filename,
			&post.FilenameOriginal, &post.FileChecksum, &post.Filesize, &post.ImageW,
			&post.ImageH, &post.ThumbW, &post.ThumbH, &post.IP, &post.Tag, &post.Timestamp,
			&post.Autosage, &post.PosterAuthority, &post.DeletedTimestamp, &post.Bumped,
			&post.Stickied, &post.Locked, &post.Reviewed, &post.Sillytag,
		); err != nil {
			handleError(0, customError(err))
			return
		}
		posts = append(posts, post)
	}
	return
}

// TODO: replace where with a map[string]interface{} like getBoardsArr()
func getSectionArr(where string) (sections []interface{}, err error) {
	if where == "" {
		where = "1"
	}
	rows, err := querySQL("SELECT * FROM `" + config.DBprefix + "sections` WHERE " + where + " ORDER BY `order`")
	defer closeRows(rows)
	if err != nil {
		errorLog.Print(err.Error())
		return
	}

	for rows.Next() {
		section := new(BoardSectionsTable)
		section.IName = "section"

		if err = rows.Scan(&section.ID, &section.Order, &section.Hidden, &section.Name, &section.Abbreviation); err != nil {
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

func getFormattedFilesize(size int) string {
	if size < 1000 {
		return fmt.Sprintf("%fB", size)
	} else if size <= 100000 {
		return fmt.Sprintf("%fKB", size/1024)
	} else if size <= 100000000 {
		return fmt.Sprintf("%fMB", size/1024/1024)
	}
	return fmt.Sprintf("%0.2fGB", size/1024/1024/1024)
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
		file, line, _ := getMetaInfo(1)
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

// paginate returns a 2d array of a specified interface from a 1d array passed in,
//	with a specified number of values per array in the 2d array.
// interface_length is the number of interfaces per array in the 2d array (e.g, threads per page)
// interf is the array of interfaces to be split up.
func paginate(interfaceLength int, interf []interface{}) [][]interface{} {
	// paginated_interfaces = the finished interface array
	// num_arrays = the current number of arrays (before remainder overflow)
	// interfaces_remaining = if greater than 0, these are the remaining interfaces
	// 		that will be added to the super-interface

	var paginatedInterfaces [][]interface{}
	numArrays := len(interf) / interfaceLength
	interfacesRemaining := len(interf) % interfaceLength
	//paginated_interfaces = append(paginated_interfaces, interf)
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

// sanitize/escape HTML strings in a post. This should be run immediately before
// the post is inserted into the database
func sanitizePost(post *PostTable) {
	post.Name = html.EscapeString(post.Name)
	post.Email = html.EscapeString(post.Email)
	post.Subject = html.EscapeString(post.Subject)
	post.Password = html.EscapeString(post.Password)
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
func checkAkismetAPIKey() {
	resp, err := http.PostForm("https://rest.akismet.com/1.1/verify-key", url.Values{"key": {config.AkismetAPIKey}, "blog": {"http://" + config.SiteDomain}})
	if err != nil {
		handleError(1, err.Error())
	}
	defer func() {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		handleError(1, err.Error())
	}
	if string(body) == "invalid" {
		// This should disable the Akismet checks if the API key is not valid.
		errorLog.Print("Akismet API key is invalid, Akismet spam protection will be disabled.")
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
			handleError(1, err.Error())
			return "other_failure"
		}
		req.Header.Set("User-Agent", "gochan/1.0 | Akismet/0.1")
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := client.Do(req)
		if err != nil {
			handleError(1, err.Error())
			return "other_failure"
		}
		defer func() {
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
		}()
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

func makePostJSON(post PostTable, anonymous string) (postObj PostJSON) {
	var filename string
	var fileExt string
	var origFilename string

	// Separate out the extension from the filenames
	if post.Filename != "deleted" && post.Filename != "" {
		extStart := strings.LastIndex(post.Filename, ".")
		fileExt = post.Filename[extStart:]

		origExtStart := strings.LastIndex(post.FilenameOriginal, fileExt)
		origFilename = post.FilenameOriginal[:origExtStart]
		filename = post.Filename[:extStart]
	}

	postObj = PostJSON{ID: post.ID, ParentID: post.ParentID, Subject: post.Subject, Message: post.MessageHTML,
		Name: post.Name, Timestamp: post.Timestamp.Unix(), Bumped: post.Bumped.Unix(),
		ThumbWidth: post.ThumbW, ThumbHeight: post.ThumbH, ImageWidth: post.ImageW, ImageHeight: post.ImageH,
		FileSize: post.Filesize, OrigFilename: origFilename, Extension: fileExt, Filename: filename, FileChecksum: post.FileChecksum}

	// Handle Anonymous
	if post.Name == "" {
		postObj.Name = anonymous
	}

	// If we have a Tripcode, prepend a !
	if post.Tripcode != "" {
		postObj.Tripcode = "!" + post.Tripcode
	}
	return
}
