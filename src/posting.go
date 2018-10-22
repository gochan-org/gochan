// functions for handling posting, uploading, and post/thread/board page building

package main

import (
	"bytes"
	"crypto/md5"
	"database/sql"
	"errors"
	"fmt"
	"html"
	"image"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/aquilax/tripcode"
	"github.com/disintegration/imaging"
)

const (
	whitespaceMatch = "[\000-\040]"
	gt              = "&gt;"
	yearInSeconds   = 31536000
)

var (
	allSections []interface{}
	allBoards   []interface{}
)

// bumps the given thread on the given board and returns true if there were no errors
func bumpThread(postID, boardID int) error {
	_, err := execSQL("UPDATE `"+config.DBprefix+"posts` SET `bumped` = ? WHERE `id` = ? AND `boardid` = ?",
		time.Now(), postID, boardID,
	)

	return err
}

// Checks check poster's name/tripcode/file checksum (from PostTable post) for banned status
// returns ban table if the user is banned or errNotBanned if they aren't
func getBannedStatus(request *http.Request) (BanlistTable, error) {
	var banEntry BanlistTable

	formName := request.FormValue("postname")
	var tripcode string
	if formName != "" {
		parsedName := parseName(formName)
		tripcode += parsedName["name"]
		if tc, ok := parsedName["tripcode"]; ok {
			tripcode += "!" + tc
		}
	}
	ip := getRealIP(request)

	var filename string
	var checksum string
	file, fileHandler, err := request.FormFile("imagefile")
	defer func() {
		if file != nil {
			file.Close()
		}
	}()
	if err == nil {
		html.EscapeString(fileHandler.Filename)
		if data, err2 := ioutil.ReadAll(file); err2 == nil {
			checksum = fmt.Sprintf("%x", md5.Sum(data))
		}
	}

	in := []interface{}{ip}
	query := "SELECT `id`,`ip`,`name`,`boards`,`timestamp`,`expires`,`permaban`,`reason`,`type`,`appeal_at`,`can_appeal` FROM `" + config.DBprefix + "banlist` WHERE `ip` = ? "

	if tripcode != "" {
		in = append(in, tripcode)
		query += "OR `name` = ? "
	}
	if filename != "" {
		in = append(in, filename)
		query += "OR `filename` = ? "
	}
	if checksum != "" {
		in = append(in, checksum)
		query += "OR `file_checksum` = ? "
	}
	query += " ORDER BY `id` DESC LIMIT 1"

	err = queryRowSQL(query, in, []interface{}{
		&banEntry.ID, &banEntry.IP, &banEntry.Name, &banEntry.Boards, &banEntry.Timestamp,
		&banEntry.Expires, &banEntry.Permaban, &banEntry.Reason, &banEntry.Type,
		&banEntry.AppealAt, &banEntry.CanAppeal},
	)
	return banEntry, err
}

func isBanned(ban BanlistTable, board string) bool {
	if ban.Boards == "" && (ban.Expires.After(time.Now()) || ban.Permaban) {
		return true
	}
	boardsArr := strings.Split(ban.Boards, ",")
	for _, b := range boardsArr {
		if b == board && (ban.Expires.After(time.Now()) || ban.Permaban) {
			return true
		}
	}

	return false
}

func sinceLastPost(post *PostTable) int {
	var lastPostTime time.Time
	if err := queryRowSQL("SELECT `timestamp` FROM `"+config.DBprefix+"posts` WHERE `ip` = '?' ORDER BY `timestamp` DESC LIMIT 1",
		[]interface{}{post.IP},
		[]interface{}{&lastPostTime},
	); err == sql.ErrNoRows {
		// no posts by that IP.
		return -1
	}
	return int(time.Since(lastPostTime).Seconds())
}

func createImageThumbnail(image_obj image.Image, size string) image.Image {
	var thumbWidth int
	var thumbHeight int

	switch size {
	case "op":
		thumbWidth = config.ThumbWidth
		thumbHeight = config.ThumbHeight
	case "reply":
		thumbWidth = config.ThumbWidth_reply
		thumbHeight = config.ThumbHeight_reply
	case "catalog":
		thumbWidth = config.ThumbWidth_catalog
		thumbHeight = config.ThumbHeight_catalog
	}
	old_rect := image_obj.Bounds()
	if thumbWidth >= old_rect.Max.X && thumbHeight >= old_rect.Max.Y {
		return image_obj
	}

	thumbW, thumbH := getThumbnailSize(old_rect.Max.X, old_rect.Max.Y, size)
	image_obj = imaging.Resize(image_obj, thumbW, thumbH, imaging.CatmullRom) // resize to 600x400 px using CatmullRom cubic filter
	return image_obj
}

func createVideoThumbnail(video, thumb string, size int) error {
	sizeStr := strconv.Itoa(size)
	outputBytes, err := exec.Command("ffmpeg", "-y", "-itsoffset", "-1", "-i", video, "-vframes", "1", "-filter:v", "scale='min("+sizeStr+"\\, "+sizeStr+"):-1'", thumb).CombinedOutput()
	println(2, "ffmpeg output: \n"+string(outputBytes))
	if err != nil {
		outputStringArr := strings.Split(string(outputBytes), "\n")
		if len(outputStringArr) > 1 {
			outputString := outputStringArr[len(outputStringArr)-2]
			err = errors.New(outputString)
		}
	}
	return err
}

func getVideoInfo(path string) (map[string]int, error) {
	vidInfo := make(map[string]int)

	outputBytes, err := exec.Command("ffprobe", "-v quiet", "-show_format", "-show_streams", path).CombinedOutput()
	if err == nil && outputBytes != nil {
		outputStringArr := strings.Split(string(outputBytes), "\n")
		for _, line := range outputStringArr {
			lineArr := strings.Split(line, "=")
			if len(lineArr) < 2 {
				continue
			}

			if lineArr[0] == "width" || lineArr[0] == "height" || lineArr[0] == "size" {
				value, _ := strconv.Atoi(lineArr[1])
				vidInfo[lineArr[0]] = value
			}
		}
	}
	return vidInfo, err
}

func getNewFilename() string {
	now := time.Now().Unix()
	rand.Seed(now)
	return strconv.Itoa(int(now)) + strconv.Itoa(rand.Intn(98)+1)
}

// find out what out thumbnail's width and height should be, partially ripped from Kusaba X
func getThumbnailSize(w int, h int, size string) (newWidth int, newHeight int) {
	var thumbWidth int
	var thumbHeight int

	switch {
	case size == "op":
		thumbWidth = config.ThumbWidth
		thumbHeight = config.ThumbHeight
	case size == "reply":
		thumbWidth = config.ThumbWidth_reply
		thumbHeight = config.ThumbHeight_reply
	case size == "catalog":
		thumbWidth = config.ThumbWidth_catalog
		thumbHeight = config.ThumbHeight_catalog
	}
	if w == h {
		newWidth = thumbWidth
		newHeight = thumbHeight
	} else {
		var percent float32
		if w > h {
			percent = float32(thumbWidth) / float32(w)
		} else {
			percent = float32(thumbHeight) / float32(h)
		}
		newWidth = int(float32(w) * percent)
		newHeight = int(float32(h) * percent)
	}
	return
}

func parseName(name string) map[string]string {
	parsed := make(map[string]string)
	if !strings.Contains(name, "#") {
		parsed["name"] = name
		parsed["tripcode"] = ""
	} else if strings.Index(name, "#") == 0 {
		parsed["tripcode"] = tripcode.Tripcode(name[1:])
	} else if strings.Index(name, "#") > 0 {
		postNameArr := strings.SplitN(name, "#", 2)
		parsed["name"] = postNameArr[0]
		parsed["tripcode"] = tripcode.Tripcode(postNameArr[1])
	}
	return parsed
}

// inserts prepared post object into the SQL table so that it can be rendered
func insertPost(post PostTable, bump bool) (sql.Result, error) {
	result, err := execSQL(
		"INSERT INTO `"+config.DBprefix+"posts` "+
			"(`boardid`,`parentid`,`name`,`tripcode`,`email`,`subject`,`message`,`message_raw`,`password`,`filename`,`filename_original`,`file_checksum`,`filesize`,`image_w`,`image_h`,`thumb_w`,`thumb_h`,`ip`,`tag`,`timestamp`,`autosage`,`poster_authority`,`deleted_timestamp`,`bumped`,`stickied`,`locked`,`reviewed`,`sillytag`)"+
			"VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)",
		post.BoardID, post.ParentID, post.Name, post.Tripcode, post.Email,
		post.Subject, post.MessageHTML, post.MessageText, post.Password,
		post.Filename, post.FilenameOriginal, post.FileChecksum, post.Filesize,
		post.ImageW, post.ImageH, post.ThumbW, post.ThumbH, post.IP, post.Tag,
		post.Timestamp, post.Autosage, post.PosterAuthority, post.DeletedTimestamp,
		post.Bumped, post.Stickied, post.Locked, post.Reviewed, post.Sillytag,
	)

	if err != nil {
		return result, err
	}

	// Bump parent post if requested.
	if post.ParentID != 0 && bump {
		err = bumpThread(post.ParentID, post.BoardID)
		if err != nil {
			return nil, err
		}
	}
	return result, err
}

// called when a user accesses /post. Parse form data, then insert and build
func makePost(writer http.ResponseWriter, request *http.Request) {
	startTime := benchmarkTimer("makePost", time.Now(), true)
	var maxMessageLength int
	var post PostTable
	domain := request.Host
	var formName string
	var nameCookie string
	var formEmail string

	// fix new cookie domain for when you use a port number
	chopPortNumRegex := regexp.MustCompile(`(.+|\w+):(\d+)$`)
	domain = chopPortNumRegex.Split(domain, -1)[0]

	post.ParentID, _ = strconv.Atoi(request.FormValue("threadid"))
	post.BoardID, _ = strconv.Atoi(request.FormValue("boardid"))

	var emailCommand string
	formName = request.FormValue("postname")
	parsedName := parseName(formName)
	post.Name = parsedName["name"]
	post.Tripcode = parsedName["tripcode"]

	formEmail = request.FormValue("postemail")
	http.SetCookie(writer, &http.Cookie{Name: "email", Value: formEmail, Path: "/", Domain: domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(yearInSeconds))), MaxAge: yearInSeconds})

	if !strings.Contains(formEmail, "noko") && !strings.Contains(formEmail, "sage") {
		post.Email = formEmail
	} else if strings.Index(formEmail, "#") > 1 {
		formEmailArr := strings.SplitN(formEmail, "#", 2)
		post.Email = formEmailArr[0]
		emailCommand = formEmailArr[1]
	} else if formEmail == "noko" || formEmail == "sage" {
		emailCommand = formEmail
		post.Email = ""
	}

	post.Subject = request.FormValue("postsubject")
	post.MessageText = strings.Trim(request.FormValue("postmsg"), "\r\n")

	if err := queryRowSQL("SELECT `max_message_length` from `"+config.DBprefix+"boards` WHERE `id` = ?",
		[]interface{}{post.BoardID},
		[]interface{}{&maxMessageLength},
	); err != nil {
		serveErrorPage(writer, handleError(0, "Error getting board info: "+err.Error()))
		return
	}

	if len(post.MessageText) > maxMessageLength {
		serveErrorPage(writer, "Post body is too long")
		return
	}
	post.MessageHTML = formatMessage(post.MessageText)
	post.Password = md5Sum(request.FormValue("postpassword"))

	// Reverse escapes
	nameCookie = strings.Replace(formName, "&amp;", "&", -1)
	nameCookie = strings.Replace(nameCookie, "\\&#39;", "'", -1)
	nameCookie = strings.Replace(url.QueryEscape(nameCookie), "+", "%20", -1)

	// add name and email cookies that will expire in a year (31536000 seconds)
	http.SetCookie(writer, &http.Cookie{Name: "name", Value: nameCookie, Path: "/", Domain: domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(yearInSeconds))), MaxAge: yearInSeconds})
	http.SetCookie(writer, &http.Cookie{Name: "password", Value: request.FormValue("postpassword"), Path: "/", Domain: domain, RawExpires: getSpecificSQLDateTime(time.Now().Add(time.Duration(yearInSeconds))), MaxAge: yearInSeconds})

	post.IP = getRealIP(request)
	post.Timestamp = time.Now()
	post.PosterAuthority = getStaffRank(request)
	post.Bumped = time.Now()
	post.Stickied = request.FormValue("modstickied") == "on"
	post.Locked = request.FormValue("modlocked") == "on"

	//post has no referrer, or has a referrer from a different domain, probably a spambot
	if !validReferrer(request) {
		accessLog.Print("Rejected post from possible spambot @ " + post.IP)
		return
	}

	switch checkPostForSpam(post.IP, request.Header["User-Agent"][0], request.Referer(),
		post.Name, post.Email, post.MessageText) {
	case "discard":
		serveErrorPage(writer, "Your post looks like spam.")
		accessLog.Print("Akismet recommended discarding post from: " + post.IP)
		return
	case "spam":
		serveErrorPage(writer, "Your post looks like spam.")
		accessLog.Print("Akismet suggested post is spam from " + post.IP)
		return
	default:
	}

	file, handler, err := request.FormFile("imagefile")
	defer func() {
		if file != nil {
			file.Close()
		}
	}()
	if err != nil || handler.Size == 0 {
		// no file was uploaded
		post.Filename = ""
		accessLog.Print("Receiving post from " + post.IP + ", referred from: " + request.Referer())
	} else {
		data, err := ioutil.ReadAll(file)
		if err != nil {
			serveErrorPage(writer, handleError(1, "Couldn't read file: "+err.Error()))
		} else {
			post.FilenameOriginal = html.EscapeString(handler.Filename)
			filetype := getFileExtension(post.FilenameOriginal)
			thumbFiletype := filetype
			if thumbFiletype == "gif" || thumbFiletype == "webm" {
				thumbFiletype = "jpg"
			}

			post.Filename = getNewFilename() + "." + getFileExtension(post.FilenameOriginal)
			boardArr, _ := getBoardArr(map[string]interface{}{"id": request.FormValue("boardid")}, "")
			if len(boardArr) == 0 {
				serveErrorPage(writer, "No boards have been created yet")
				return
			}
			_boardDir, _ := getBoardArr(map[string]interface{}{"id": request.FormValue("boardid")}, "")
			boardDir := _boardDir[0].Dir
			filePath := path.Join(config.DocumentRoot, "/"+boardDir+"/src/", post.Filename)
			thumbPath := path.Join(config.DocumentRoot, "/"+boardDir+"/thumb/", strings.Replace(post.Filename, "."+filetype, "t."+thumbFiletype, -1))
			catalogThumbPath := path.Join(config.DocumentRoot, "/"+boardDir+"/thumb/", strings.Replace(post.Filename, "."+filetype, "c."+thumbFiletype, -1))

			if err := ioutil.WriteFile(filePath, data, 0777); err != nil {
				handleError(0, "Couldn't write file \""+post.Filename+"\""+err.Error())
				serveErrorPage(writer, "Couldn't write file \""+post.FilenameOriginal+"\"")
				return
			}

			// Calculate image checksum
			post.FileChecksum = fmt.Sprintf("%x", md5.Sum(data))

			var allowsVids bool
			if err = queryRowSQL("SELECT `embeds_allowed` FROM `"+config.DBprefix+"boards` WHERE `id` = ? LIMIT 1",
				[]interface{}{post.BoardID},
				[]interface{}{&allowsVids},
			); err != nil {
				serveErrorPage(writer, handleError(1, "Couldn't get board info: "+err.Error()))
				return
			}

			if filetype == "webm" {
				if !allowsVids || !config.AllowVideoUploads {
					serveErrorPage(writer, "Video uploading is not currently enabled for this board.")
					os.Remove(filePath)
					return
				}

				accessLog.Print("Receiving post with video: " + handler.Filename + " from " + request.RemoteAddr + ", referrer: " + request.Referer())
				if post.ParentID == 0 {
					err := createVideoThumbnail(filePath, thumbPath, config.ThumbWidth)
					if err != nil {
						serveErrorPage(writer, handleError(1, err.Error()))
						return
					}
				} else {
					err := createVideoThumbnail(filePath, thumbPath, config.ThumbWidth_reply)
					if err != nil {
						serveErrorPage(writer, handleError(1, err.Error()))
						return
					}
				}

				if err := createVideoThumbnail(filePath, catalogThumbPath, config.ThumbWidth_catalog); err != nil {
					serveErrorPage(writer, handleError(1, err.Error()))
					return
				}

				outputBytes, err := exec.Command("ffprobe", "-v", "quiet", "-show_format", "-show_streams", filePath).CombinedOutput()
				if err != nil {
					serveErrorPage(writer, handleError(1, "Error getting video info: "+err.Error()))
					return
				}
				if err == nil && outputBytes != nil {
					outputStringArr := strings.Split(string(outputBytes), "\n")
					for _, line := range outputStringArr {
						lineArr := strings.Split(line, "=")
						if len(lineArr) < 2 {
							continue
						}
						value, _ := strconv.Atoi(lineArr[1])
						switch lineArr[0] {
						case "width":
							post.ImageW = value
						case "height":
							post.ImageH = value
						case "size":
							post.Filesize = value
						}
					}
					if post.ParentID == 0 {
						post.ThumbW, post.ThumbH = getThumbnailSize(post.ImageW, post.ImageH, "op")
					} else {
						post.ThumbW, post.ThumbH = getThumbnailSize(post.ImageW, post.ImageH, "reply")
					}
				}

			} else {
				// Attempt to load uploaded file with imaging library
				img, err := imaging.Open(filePath)
				if err != nil {
					os.Remove(filePath)
					handleError(1, "Couldn't open uploaded file \""+post.Filename+"\""+err.Error())
					serveErrorPage(writer, "Upload filetype not supported")
					return
				} else {
					// Get image filesize
					stat, err := os.Stat(filePath)
					if err != nil {
						serveErrorPage(writer, handleError(1, "Couldn't get image filesize: "+err.Error()))
						return
					} else {
						post.Filesize = int(stat.Size())
					}

					// Get image width and height, as well as thumbnail width and height
					post.ImageW = img.Bounds().Max.X
					post.ImageH = img.Bounds().Max.Y
					if post.ParentID == 0 {
						post.ThumbW, post.ThumbH = getThumbnailSize(post.ImageW, post.ImageH, "op")
					} else {
						post.ThumbW, post.ThumbH = getThumbnailSize(post.ImageW, post.ImageH, "reply")
					}

					accessLog.Print("Receiving post with image: " + handler.Filename + " from " + request.RemoteAddr + ", referrer: " + request.Referer())

					if request.FormValue("spoiler") == "on" {
						// If spoiler is enabled, symlink thumbnail to spoiler image
						if _, err := os.Stat(path.Join(config.DocumentRoot, "spoiler.png")); err != nil {
							serveErrorPage(writer, "missing /spoiler.png")
							return
						} else {
							err = syscall.Symlink(path.Join(config.DocumentRoot, "spoiler.png"), thumbPath)
							if err != nil {
								serveErrorPage(writer, err.Error())
								return
							}
						}
					} else if config.ThumbWidth >= post.ImageW && config.ThumbHeight >= post.ImageH {
						// If image fits in thumbnail size, symlink thumbnail to original
						post.ThumbW = img.Bounds().Max.X
						post.ThumbH = img.Bounds().Max.Y
						if err := syscall.Symlink(filePath, thumbPath); err != nil {
							serveErrorPage(writer, err.Error())
							return
						}
					} else {
						var thumbnail image.Image
						var catalogThumbnail image.Image
						if post.ParentID == 0 {
							// If this is a new thread, generate thumbnail and catalog thumbnail
							thumbnail = createImageThumbnail(img, "op")
							catalogThumbnail = createImageThumbnail(img, "catalog")
							if err = imaging.Save(catalogThumbnail, catalogThumbPath); err != nil {
								serveErrorPage(writer, handleError(1, "Couldn't generate catalog thumbnail: "+err.Error()))
								return
							}
						} else {
							thumbnail = createImageThumbnail(img, "reply")
						}
						if err = imaging.Save(thumbnail, thumbPath); err != nil {
							serveErrorPage(writer, handleError(1, "Couldn't save thumbnail: "+err.Error()))
							return
						}
					}
				}
			}
		}
	}

	if strings.TrimSpace(post.MessageText) == "" && post.Filename == "" {
		serveErrorPage(writer, "Post must contain a message if no image is uploaded.")
		return
	}

	postDelay := sinceLastPost(&post)
	if postDelay > -1 {
		if post.ParentID == 0 && postDelay < config.NewThreadDelay {
			serveErrorPage(writer, "Please wait before making a new thread.")
			return
		} else if post.ParentID > 0 && postDelay < config.ReplyDelay {
			serveErrorPage(writer, "Please wait before making a reply.")
			return
		}
	}

	banStatus, err := getBannedStatus(request)
	if err != nil && err != sql.ErrNoRows {
		handleError(1, "Error in getBannedStatus: "+err.Error())
		serveErrorPage(writer, err.Error())
		return
	}

	boards, _ := getBoardArr(nil, "")

	if isBanned(banStatus, boards[post.BoardID-1].Dir) {
		var banpage_buffer bytes.Buffer
		var banpage_html string
		banpage_buffer.Write([]byte(""))
		if err = banpage_tmpl.Execute(&banpage_buffer, map[string]interface{}{
			"config": config, "ban": banStatus, "banBoards": boards[post.BoardID-1].Dir,
		}); err != nil {
			fmt.Fprintf(writer, banpage_html+handleError(1, err.Error())+"\n</body>\n</html>")
			return
		}
		fmt.Fprintf(writer, banpage_buffer.String())
		return
	}

	post.Sanitize()
	result, err := insertPost(post, emailCommand != "sage")
	if err != nil {
		serveErrorPage(writer, handleError(1, err.Error()))
		return
	}
	postid, _ := result.LastInsertId()
	post.ID = int(postid)

	// rebuild the board page
	buildBoards(false, post.BoardID)
	buildFrontPage()

	if emailCommand == "noko" {
		if post.ParentID == 0 {
			http.Redirect(writer, request, config.SiteWebfolder+boards[post.BoardID-1].Dir+"/res/"+strconv.Itoa(post.ID)+".html", http.StatusFound)
		} else {
			http.Redirect(writer, request, config.SiteWebfolder+boards[post.BoardID-1].Dir+"/res/"+strconv.Itoa(post.ParentID)+".html#"+strconv.Itoa(post.ID), http.StatusFound)
		}
	} else {
		http.Redirect(writer, request, config.SiteWebfolder+boards[post.BoardID-1].Dir+"/", http.StatusFound)
	}
	benchmarkTimer("makePost", startTime, false)
}

func formatMessage(message string) string {
	message = bbcompiler.Compile(message)
	// prepare each line to be formatted
	postLines := strings.Split(message, "<br>")
	for i, line := range postLines {
		trimmedLine := strings.TrimSpace(line)
		lineWords := strings.Split(trimmedLine, " ")
		isGreentext := false // if true, append </span> to end of line
		for w, word := range lineWords {
			if strings.LastIndex(word, gt+gt) == 0 {
				//word is a backlink
				if _, err := strconv.Atoi(word[8:]); err == nil {
					// the link is in fact, a valid int
					var boardDir string
					var linkParent int

					if err = queryRowSQL("SELECT `dir`,`parentid` FROM "+config.DBprefix+"posts,"+config.DBprefix+"boards WHERE "+config.DBprefix+"posts.id = ?",
						[]interface{}{word[8:]},
						[]interface{}{&boardDir, &linkParent},
					); err != nil {
						handleError(1, customError(err))
					}

					// get post board dir
					if boardDir == "" {
						lineWords[w] = "<a href=\"javascript:;\"><strike>" + word + "</strike></a>"
					} else if linkParent == 0 {
						lineWords[w] = "<a href=\"" + config.SiteWebfolder + boardDir + "/res/" + word[8:] + ".html\">" + word + "</a>"
					} else {
						lineWords[w] = "<a href=\"" + config.SiteWebfolder + boardDir + "/res/" + strconv.Itoa(linkParent) + ".html#" + word[8:] + "\">" + word + "</a>"
					}
				}
			} else if strings.Index(word, gt) == 0 && w == 0 {
				// word is at the beginning of a line, and is greentext
				isGreentext = true
				lineWords[w] = "<span class=\"greentext\">" + word
			}
		}
		line = strings.Join(lineWords, " ")
		if isGreentext {
			line += "</span>"
		}
		postLines[i] = line
	}
	return strings.Join(postLines, "<br />")
}

func bannedForever(ban BanlistTable) bool {
	return ban.Permaban && !ban.CanAppeal && ban.Type == 3 && ban.Boards == ""
}

func banHandler(writer http.ResponseWriter, request *http.Request) {
	appealMsg := request.FormValue("appealmsg")
	banStatus, err := getBannedStatus(request)

	if appealMsg != "" {
		if bannedForever(banStatus) {
			fmt.Fprint(writer, "No.")
			return
		}
		escapedMsg := html.EscapeString(appealMsg)
		if _, err = execSQL("INSERT INTO `"+config.DBprefix+"appeals` (`ban`,`message`) VALUES(?,?)",
			banStatus.ID, escapedMsg,
		); err != nil {
			serveErrorPage(writer, err.Error())
		}
		fmt.Fprint(writer,
			"Appeal sent. It will (hopefully) be read by a staff member. check "+config.SiteWebfolder+"banned occasionally for a response",
		)
		return
	}

	if err != nil && err != sql.ErrNoRows {
		handleError(1, "Error in getBannedStatus: "+err.Error())
		serveErrorPage(writer, err.Error())
		return
	}

	var banpageBuffer bytes.Buffer

	banpageBuffer.Write([]byte(""))
	if err = banpage_tmpl.Execute(&banpageBuffer, map[string]interface{}{
		"config": config, "ban": banStatus, "banBoards": banStatus.Boards, "post": PostTable{},
	}); err != nil {
		fmt.Fprintf(writer, handleError(1, err.Error())+"\n</body>\n</html>")
		return
	}
	fmt.Fprintf(writer, banpageBuffer.String())
}
