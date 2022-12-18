package posting

import (
	"errors"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/serverutil"
)

const (
	yearInSeconds = 31536000
	maxFormBytes  = 50000000
)

var (
	ErrorPostTooLong = errors.New("post is too long")
)

// MakePost is called when a user accesses /post. Parse form data, then insert and build
func MakePost(writer http.ResponseWriter, request *http.Request) {
	request.ParseMultipartForm(maxFormBytes)
	ip := gcutil.GetRealIP(request)
	errEv := gcutil.LogError(nil).
		Str("IP", ip)
	infoEv := gcutil.LogInfo().
		Str("IP", ip)
	defer func() {
		errEv.Discard()
		infoEv.Discard()
	}()
	var post gcsql.Post
	var formName string
	var nameCookie string
	var formEmail string

	systemCritical := config.GetSystemCriticalConfig()

	if request.Method == "GET" {
		infoEv.Msg("Invalid request (expected POST, not GET)")
		http.Redirect(writer, request, systemCritical.WebRoot, http.StatusFound)
		return
	}

	if request.FormValue("doappeal") != "" {
		handleAppeal(writer, request, errEv)
		return
	}

	wantsJSON := serverutil.IsRequestingJSON(request)
	post.IP = gcutil.GetRealIP(request)
	var err error
	threadidStr := request.FormValue("threadid")
	// to avoid potential hiccups, we'll just treat the "threadid" form field as the OP ID and convert it internally
	// to the real thread ID
	var opID int
	if threadidStr != "" {
		// post is a reply
		if opID, err = strconv.Atoi(threadidStr); err != nil {
			errEv.Err(err).
				Str("opIDstr", threadidStr).
				Caller().Msg("Invalid threadid value")
			serverutil.ServeError(writer, "Invalid form data (invalid threadid)", wantsJSON, map[string]interface{}{
				"threadid": threadidStr,
			})
			return
		}
		if opID > 0 {
			if post.ThreadID, err = gcsql.GetTopPostThreadID(opID); err != nil {
				errEv.Err(err).
					Int("opID", opID).
					Caller().Send()
				serverutil.ServeError(writer, err.Error(), wantsJSON, map[string]interface{}{
					"opID": opID,
				})
			}
		}
	}

	boardidStr := request.FormValue("boardid")
	boardID, err := strconv.Atoi(boardidStr)
	if err != nil {
		errEv.Str("boardid", boardidStr).Caller().Msg("Invalid boardid value")
		serverutil.ServeError(writer, "Invalid form data (invalid boardid)", wantsJSON, map[string]interface{}{
			"boardid": boardidStr,
		})
		return
	}
	postBoard, err := gcsql.GetBoardFromID(boardID)
	if err != nil {
		errEv.Err(err).Caller().
			Int("boardid", boardID).
			Msg("Unable to get board info")
		serverutil.ServeError(writer, "Unable to get board info", wantsJSON, map[string]interface{}{
			"boardid": boardID,
		})
		return
	}
	boardConfig := config.GetBoardConfig(postBoard.Dir)

	var emailCommand string
	formName = request.FormValue("postname")
	parsedName := gcutil.ParseName(formName)
	post.Name = parsedName["name"]
	post.Tripcode = parsedName["tripcode"]

	formEmail = request.FormValue("postemail")

	http.SetCookie(writer, &http.Cookie{
		Name:   "email",
		Value:  formEmail,
		MaxAge: yearInSeconds,
	})

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
	post.MessageRaw = strings.TrimSpace(request.FormValue("postmsg"))
	if len(post.MessageRaw) > postBoard.MaxMessageLength {
		errEv.
			Int("messageLength", len(post.MessageRaw)).
			Int("maxMessageLength", postBoard.MaxMessageLength).Send()
		serverutil.ServeError(writer, "Message is too long", wantsJSON, map[string]interface{}{
			"messageLength": len(post.MessageRaw),
			"boardid":       boardID,
		})
		return
	}

	if post.MessageRaw, err = ApplyWordFilters(post.MessageRaw, postBoard.Dir); err != nil {
		errEv.Err(err).Caller().Msg("Error formatting post")
		serverutil.ServeError(writer, "Error formatting post: "+err.Error(), wantsJSON, map[string]interface{}{
			"boardDir": postBoard.Dir,
		})
		return
	}

	post.Message = FormatMessage(post.MessageRaw, postBoard.Dir)
	password := request.FormValue("postpassword")
	if password == "" {
		password = gcutil.RandomString(8)
	}
	post.Password = gcutil.Md5Sum(password)

	// Reverse escapes
	nameCookie = strings.Replace(formName, "&amp;", "&", -1)
	nameCookie = strings.Replace(nameCookie, "\\&#39;", "'", -1)
	nameCookie = strings.Replace(url.QueryEscape(nameCookie), "+", "%20", -1)

	// add name and email cookies that will expire in a year (31536000 seconds)
	http.SetCookie(writer, &http.Cookie{
		Name:   "name",
		Value:  nameCookie,
		MaxAge: yearInSeconds,
	})
	http.SetCookie(writer, &http.Cookie{
		Name:   "password",
		Value:  password,
		MaxAge: yearInSeconds,
	})

	post.CreatedOn = time.Now()
	// post.PosterAuthority = getStaffRank(request)
	// bumpedTimestamp := time.Now()
	// isSticky := request.FormValue("modstickied") == "on"
	// isLocked := request.FormValue("modlocked") == "on"

	//post has no referrer, or has a referrer from a different domain, probably a spambot
	if !serverutil.ValidReferer(request) {
		gcutil.LogWarning().
			Str("spam", "badReferer").
			Str("IP", post.IP).
			Msg("Rejected post from possible spambot")
		serverutil.ServeError(writer, "Your post looks like spam", wantsJSON, nil)
		return
	}

	akismetResult := serverutil.CheckPostForSpam(
		post.IP, request.Header.Get("User-Agent"), request.Referer(),
		post.Name, post.Email, post.MessageRaw,
	)
	logEvent := gcutil.LogInfo().
		Str("User-Agent", request.Header.Get("User-Agent")).
		Str("IP", post.IP)
	switch akismetResult {
	case "discard":
		logEvent.Str("akismet", "discard").Send()
		serverutil.ServeError(writer, "Your post looks like spam.", wantsJSON, nil)
		return
	case "spam":
		logEvent.Str("akismet", "spam").Send()
		serverutil.ServeError(writer, "Your post looks like spam.", wantsJSON, nil)
		return
	default:
		logEvent.Discard()
	}

	var delay int
	var tooSoon bool
	if threadidStr == "" || threadidStr == "0" {
		// creating a new thread
		delay, err = gcsql.SinceLastThread(post.IP)
		tooSoon = delay < boardConfig.Cooldowns.NewThread
	} else {
		delay, err = gcsql.SinceLastPost(post.IP)
		tooSoon = delay < boardConfig.Cooldowns.Reply
	}
	if err != nil {
		errEv.Err(err).Caller().Str("boardDir", postBoard.Dir).Msg("Unable to check post cooldown")
		serverutil.ServeError(writer, "Error checking post cooldown: "+err.Error(), wantsJSON, map[string]interface{}{
			"boardDir": postBoard.Dir,
		})
		return
	}
	if tooSoon {
		errEv.Int("delay", delay).Msg("Rejecting post (user must wait before making another post)")
		serverutil.ServeError(writer, "Please wait before making a new post", wantsJSON, nil)
		return
	}

	if checkIpBan(&post, postBoard, writer, request) {
		return
	}
	if checkUsernameBan(&post, postBoard, writer, request) {
		return
	}

	post.Sanitize()

	if boardConfig.UseCaptcha {
		captchaID := request.FormValue("captchaid")
		captchaAnswer := request.FormValue("captchaanswer")
		if captchaID == "" && captchaAnswer == "" {
			// browser isn't using JS, save post data to tempPosts and show captcha
			request.Form.Add("temppostindex", strconv.Itoa(len(gcsql.TempPosts)))
			request.Form.Add("emailcmd", emailCommand)
			gcsql.TempPosts = append(gcsql.TempPosts, post)

			ServeCaptcha(writer, request)
			return
		}
	}

	boardExists := gcsql.DoesBoardExistByID(boardID)
	if !boardExists {
		serverutil.ServeErrorPage(writer, "Board does not exist (invalid boardid)")
		return
	}

	upload, gotErr := AttachUploadFromRequest(request, writer, &post, postBoard)
	if gotErr {
		// got an error receiving the upload, stop here (assuming an error page was actually shown)
		return
	}
	documentRoot := config.GetSystemCriticalConfig().DocumentRoot

	if err = post.Insert(emailCommand != "sage", postBoard.ID, false, false, false, false); err != nil {
		errEv.Err(err).Caller().
			Str("sql", "postInsertion").
			Msg("Unable to insert post")
		if upload != nil {
			filePath := path.Join(documentRoot, postBoard.Dir, "src", upload.Filename)
			thumbPath := path.Join(documentRoot, postBoard.Dir, "thumb", upload.ThumbnailPath("thumb"))
			catalogThumbPath := path.Join(documentRoot, postBoard.Dir, "thumb", upload.ThumbnailPath("catalog"))
			os.Remove(filePath)
			os.Remove(thumbPath)
			os.Remove(catalogThumbPath)
		}
		serverutil.ServeErrorPage(writer, "Unable to insert post: "+err.Error())
		return
	}

	if err = post.AttachFile(upload); err != nil {
		errEv.Err(err).Caller().
			Str("sql", "postInsertion").
			Msg("Unable to attach upload to post")
		filePath := path.Join(documentRoot, postBoard.Dir, "src", upload.Filename)
		thumbPath := path.Join(documentRoot, postBoard.Dir, "thumb", upload.ThumbnailPath("thumb"))
		catalogThumbPath := path.Join(documentRoot, postBoard.Dir, "thumb", upload.ThumbnailPath("catalog"))
		os.Remove(filePath)
		os.Remove(thumbPath)
		os.Remove(catalogThumbPath)
		serverutil.ServeErrorPage(writer, "Unable to attach upload: "+err.Error())
		return
	}

	// rebuild the board page
	if err = building.BuildBoards(false, postBoard.ID); err != nil {
		serverutil.ServeErrorPage(writer, "Error building boards: "+err.Error())
		return
	}

	if err = building.BuildFrontPage(); err != nil {
		serverutil.ServeErrorPage(writer, "Error building front page: "+err.Error())
		return
	}

	if emailCommand == "noko" {
		if post.IsTopPost {
			http.Redirect(writer, request, systemCritical.WebRoot+postBoard.Dir+"/res/"+strconv.Itoa(post.ID)+".html", http.StatusFound)
		} else {
			topPost, _ := post.TopPostID()
			http.Redirect(writer, request, systemCritical.WebRoot+postBoard.Dir+"/res/"+strconv.Itoa(topPost)+".html#"+strconv.Itoa(post.ID), http.StatusFound)
		}
	} else {
		http.Redirect(writer, request, systemCritical.WebRoot+postBoard.Dir+"/", http.StatusFound)
	}
}
