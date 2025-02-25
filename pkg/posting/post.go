package posting

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/gochan-org/gochan/pkg/building"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/events"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting/geoip"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
)

const (
	yearInSeconds = 31536000
	maxFormBytes  = 50000000
)

var (
	ErrorPostTooLong = errors.New("post is too long")
	ErrInvalidFlag   = errors.New("invalid selected flag")
)

func attachFlag(request *http.Request, post *gcsql.Post, board string, errEv *zerolog.Event) error {
	boardConfig := config.GetBoardConfig(board)
	flag := request.PostFormValue("post-flag")
	if flag != "" {
		errEv.Str("flag", flag)
	}
	var err error
	switch flag {
	case "geoip":
		if boardConfig.EnableGeoIP {
			geoipInfo, err := geoip.GetCountry(request, board, errEv)
			if err != nil {
				// GetCountry logs the error
				return err
			}
			post.Country = geoipInfo.Name
			post.Flag = strings.ToLower(geoipInfo.Flag)
		} else {
			err = ErrInvalidFlag
			errEv.Err(err).Caller().
				Msg("User selected 'geoip' on a non-geoip board")
			return err
		}
	case "":
		// "No flag"
		if !boardConfig.EnableNoFlag {
			err = ErrInvalidFlag
			errEv.Err(err).Caller().
				Msg("User submitted 'No flag' on a board without it enabled")
			return err
		}
	default:
		// custom flag
		var validFlag bool
		post.Country, validFlag = boardConfig.CheckCustomFlag(flag)
		if !validFlag {
			err = ErrInvalidFlag
			errEv.Caller().Msg("User submitted invalid custom flag")
			return err
		}
		post.Flag = flag
	}
	return nil
}

func handleRecover(writer http.ResponseWriter, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) {
	if a := recover(); a != nil {
		if writer != nil {
			writer.WriteHeader(http.StatusInternalServerError)
			server.ServeError(writer, "Internal server error", wantsJSON, nil)
		}
		errEv.Caller().
			Str("recover", fmt.Sprintf("%v", a)).
			Bytes("stack", debug.Stack()).
			Msg("Recovered from panic")
		debug.PrintStack()
	}
	errEv.Discard()
	infoEv.Discard()
}

// HandleFilterAction handles a filter's match action if the filter is not nil, and returns true if post processing should stop (an error page or ban page
// was shown)
func HandleFilterAction(filter *gcsql.Filter, post *gcsql.Post, upload *gcsql.Upload, board *gcsql.Board, writer http.ResponseWriter, request *http.Request) bool {
	if filter == nil || filter.MatchAction == "log" {
		return false
	}
	wantsJSON := serverutil.IsRequestingJSON(request)
	documentRoot := config.GetSystemCriticalConfig().DocumentRoot
	if upload != nil {
		filePath := path.Join(documentRoot, board.Dir, "thumb", upload.Filename)
		thumbPath, catalogThumbPath := uploads.GetThumbnailFilenames(
			path.Join(documentRoot, board.Dir, "thumb", upload.Filename))
		os.Remove(filePath)
		os.Remove(thumbPath)
		os.Remove(catalogThumbPath)
	}
	switch filter.MatchAction {
	case "reject":
		gcutil.LogWarning().
			Str("ip", post.IP).
			Str("userAgent", request.UserAgent()).
			Int("filterID", filter.ID).
			Msg("Post rejected by filter")
		rejectReason := filter.MatchDetail
		if rejectReason == "" {
			rejectReason = "Post rejected"
		}
		server.ServeError(writer, rejectReason, wantsJSON, nil)
	case "ban":
		// if the filter bans the user, it will be logged
		checkIpBan(post, board, writer, request)
	}
	return true
}

func setCookies(writer http.ResponseWriter, request *http.Request) {
	http.SetCookie(writer, &http.Cookie{
		Name:   "email",
		Value:  url.QueryEscape(request.PostFormValue("postemail")),
		MaxAge: yearInSeconds,
	})
	http.SetCookie(writer, &http.Cookie{
		Name:   "name",
		Value:  url.QueryEscape(request.PostFormValue("postname")),
		MaxAge: yearInSeconds,
	})
	http.SetCookie(writer, &http.Cookie{
		Name:   "password",
		Value:  url.QueryEscape(request.PostFormValue("postpassword")),
		MaxAge: yearInSeconds,
	})
}

func getEmailAndCommand(request *http.Request) (string, string) {
	formEmail := request.PostFormValue("postemail")
	if formEmail == "" || formEmail == "noko" || formEmail == "sage" {
		return "", formEmail
	}
	sepIndex := strings.LastIndex(formEmail, "#")
	if sepIndex == -1 {
		return formEmail, ""
	}
	return formEmail[:sepIndex], formEmail[sepIndex+1:]
}

func getPostFromRequest(request *http.Request, infoEv, errEv *zerolog.Event) (post *gcsql.Post, err error) {
	post = &gcsql.Post{
		IP:         gcutil.GetRealIP(request),
		Subject:    request.PostFormValue("postsubject"),
		MessageRaw: strings.TrimSpace(request.PostFormValue("postmsg")),
	}

	opIDstr := request.PostFormValue("threadid")
	// to avoid potential hiccups, we'll just treat the "threadid" form field as the OP ID and convert it internally
	// to the real thread ID
	var opID int
	if opIDstr != "" {
		// post is a reply
		if opID, err = strconv.Atoi(opIDstr); err != nil {
			errEv.Err(err).Caller().
				Str("opID", opIDstr).
				Msg("Invalid threadid value")
			return
		}
		if opID > 0 {
			gcutil.LogInt("opID", opID, infoEv, errEv)
			if post.ThreadID, err = gcsql.GetTopPostThreadID(opID); err != nil {
				errEv.Err(err).Caller().Send()
				return nil, errors.New("unable to get top post in thread")
			}
		}
	}
	post.Name, post.Tripcode = gcutil.ParseName(request.PostFormValue("postname"))
	post.Email, _ = getEmailAndCommand(request)

	password := request.PostFormValue("postpassword")
	if password == "" {
		password = gcutil.RandomString(12)
	}
	post.Password = gcutil.Md5Sum(password)
	return
}

func doFormatting(post *gcsql.Post, board *gcsql.Board, request *http.Request, errEv *zerolog.Event) (err error) {
	if len(post.MessageRaw) > board.MaxMessageLength {
		errEv.Caller().
			Int("messageLength", len(post.MessageRaw)).
			Int("maxMessageLength", board.MaxMessageLength).Send()
		return errors.New("message is too long")
	}

	if post.MessageRaw, err = ApplyWordFilters(post.MessageRaw, board.Dir); err != nil {
		errEv.Err(err).Caller().Msg("Error formatting post")
		return errors.New("unable to apply wordfilters")
	}

	_, err, recovered := events.TriggerEvent("message-pre-format", post, request)
	if recovered {
		errEv.Str("event", "message-pre-format").Msg("Recovered from a panic in an event handler")
		return errors.New("recovered from a panic in an event handler (message-pre-format)")
	}
	if err != nil {
		errEv.Err(err).Caller().
			Str("event", "message-pre-format").Send()
		return err
	}

	if post.Message, err = FormatMessage(post.MessageRaw, board.Dir); err != nil {
		errEv.Err(err).Caller().Msg("Unable to format message")
		return errors.New("unable to format message")
	}
	if err = ApplyDiceRoll(post); err != nil {
		errEv.Err(err).Caller().Msg("Error applying dice roll")
		return err
	}
	return nil
}

func getRedirectURL(post *gcsql.Post, board *gcsql.Board, request *http.Request) string {
	topPost, _ := post.TopPostID()
	_, emailCommand := getEmailAndCommand(request)

	if emailCommand == "noko" {
		if post.IsTopPost {
			return config.WebPath("/", board.Dir, "res", strconv.Itoa(post.ID)+".html")
		}
		return config.WebPath("/", board.Dir, "res", strconv.Itoa(topPost)+".html#"+strconv.Itoa(post.ID))
	}
	return config.WebPath(board.Dir)
}

// MakePost is called when a user accesses /post. Parse form data, then insert and build
func MakePost(writer http.ResponseWriter, request *http.Request) {
	request.ParseMultipartForm(maxFormBytes)
	wantsJSON := serverutil.IsRequestingJSON(request)

	infoEv, errEv := gcutil.LogRequest(request)

	refererResult, err := serverutil.CheckReferer(request)
	if err != nil {
		errEv.Err(err).Caller().Send()
		server.ServeError(writer, "Error checking referer", wantsJSON, nil)
		return
	}
	if refererResult != serverutil.InternalReferer {
		// post has no referrer, or has a referrer from a different domain, probably a spambot
		gcutil.LogWarning().
			Str("spam", "badReferer").
			Str("IP", gcutil.GetRealIP(request)).
			Str("threadID", request.PostFormValue("threadid")).
			Msg("Rejected post from possible spambot")
		server.ServeError(writer, "Your post looks like spam", wantsJSON, nil)
		return
	}

	defer handleRecover(writer, wantsJSON, infoEv, errEv)

	if request.Method == "GET" {
		infoEv.Msg("Invalid request (expected POST, not GET)")
		http.Redirect(writer, request, config.WebPath("/"), http.StatusFound)
		return
	}

	if request.PostFormValue("doappeal") != "" {
		handleAppeal(writer, request, infoEv, errEv)
		return
	}

	post, err := getPostFromRequest(request, infoEv, errEv)
	if err != nil {
		server.ServeError(writer, err.Error(), wantsJSON, nil)
		return
	}

	boardidStr := request.PostFormValue("boardid")
	boardID, err := strconv.Atoi(boardidStr)
	if err != nil {
		errEv.Str("boardid", boardidStr).Caller().Msg("Invalid boardid value")
		server.ServeError(writer, "Invalid form data (invalid boardid)", wantsJSON, map[string]any{
			"boardid": boardidStr,
		})
		return
	}
	board, err := gcsql.GetBoardFromID(boardID)
	if err != nil {
		errEv.Err(err).Caller().
			Int("boardid", boardID).
			Msg("Unable to get board info")
		server.ServeError(writer, "Unable to get board info", wantsJSON, map[string]any{
			"boardid": boardID,
		})
		return
	}
	boardConfig := config.GetBoardConfig(board.Dir)

	// do length-check, formatting, and wordfilters
	if err = doFormatting(post, board, request, errEv); err != nil {
		server.ServeError(writer, err.Error(), wantsJSON, nil)
		return
	}

	// add name, email, and password cookies that will expire in a year (31536000 seconds)
	setCookies(writer, request)

	post.CreatedOn = time.Now()
	isSticky := request.PostFormValue("modstickied") == "on"
	isLocked := request.PostFormValue("modlocked") == "on"

	if isSticky || isLocked {
		// check that the user has permission to create sticky/locked threads

		staff, err := gcsql.GetStaffFromRequest(request)
		if err != nil {
			errEv.Err(err).Caller().Msg("Unable to get staff info")
			server.ServeError(writer, "Unable to get staff info", wantsJSON, nil)
			return
		}
		if staff.Rank < 2 {
			// must be at least a moderator in order to make a sticky or locked thread
			writer.WriteHeader(http.StatusForbidden)
			server.ServeError(writer, "You do not have permission to lock or sticky threads", wantsJSON, map[string]any{
				"username": staff.Username,
				"rank":     staff.Rank,
			})
			return
		}
	}

	isCyclic := request.PostFormValue("cyclic") == "on"
	if isCyclic && !boardConfig.EnableCyclicThreads {
		writer.WriteHeader(http.StatusBadRequest)
		server.ServeError(writer, "Board does not support cyclic threads", wantsJSON, nil)
		return
	}

	var delay int
	var tooSoon bool
	if post.ThreadID == 0 {
		// creating a new thread
		delay, err = gcsql.SinceLastThread(post.IP)
		tooSoon = delay < boardConfig.Cooldowns.NewThread
	} else {
		// replying to a thread
		delay, err = gcsql.SinceLastPost(post.IP)
		tooSoon = delay < boardConfig.Cooldowns.Reply
	}
	if err != nil {
		errEv.Err(err).Caller().Str("boardDir", board.Dir).Msg("Unable to check post cooldown")
		server.ServeError(writer, "Error checking post cooldown: "+err.Error(), wantsJSON, map[string]any{
			"boardDir": board.Dir,
		})
		return
	}
	if tooSoon {
		errEv.Int("delay", delay).Msg("Rejecting post (user must wait before making another post)")
		server.ServeError(writer, "Please wait before making a new post", wantsJSON, nil)
		return
	}

	if checkIpBan(post, board, writer, request) {
		return
	}

	captchaSuccess, err := submitCaptchaResponse(request)
	if err != nil {
		errEv.Err(err).Caller().Send()
		server.ServeError(writer, "Error submitting captcha response:"+err.Error(), wantsJSON, nil)
		return
	}
	if !captchaSuccess {
		server.ServeError(writer, "Missing or invalid captcha response", wantsJSON, nil)
		errEv.Msg("Missing or invalid captcha response")
		return
	}

	if boardConfig.EnableGeoIP || len(boardConfig.CustomFlags) > 0 {
		if err = attachFlag(request, post, board.Dir, errEv); err != nil {
			server.ServeError(writer, err.Error(), wantsJSON, nil)
			return
		}
	}

	_, _, err = request.FormFile("imagefile")
	noFile := errors.Is(err, http.ErrMissingFile)
	if noFile && post.ThreadID == 0 && boardConfig.NewThreadsRequireUpload {
		errEv.Caller().Msg("New thread rejected (NewThreadsRequireUpload set in config)")
		server.ServeError(writer, "Upload required for new threads", wantsJSON, nil)
		return
	}
	if post.MessageRaw == "" && noFile {
		errEv.Caller().Msg("New post rejected (no file and message is blank)")
		server.ServeError(writer, "Your post must have an upload or a comment", wantsJSON, nil)
		return
	}

	filter, excludedFilterIDs, err := gcsql.DoNonUploadFiltering(post, boardID, request, errEv)
	if err != nil {
		server.ServeError(writer, err.Error(), wantsJSON, nil)
		return
	}
	if HandleFilterAction(filter, post, nil, board, writer, request) {
		return
	}

	upload, err := uploads.AttachUploadFromRequest(request, writer, post, board, infoEv, errEv)
	if err != nil {
		errEv.Err(err).Caller().Send()
		// got an error receiving the upload or the upload was rejected
		server.ServeError(writer, err.Error(), wantsJSON, nil)
		return
	}
	var filePath, thumbPath, catalogThumbPath string
	documentRoot := config.GetSystemCriticalConfig().DocumentRoot
	if upload != nil {
		filePath = path.Join(documentRoot, board.Dir, "src", upload.Filename)
		thumbPath, catalogThumbPath = uploads.GetThumbnailFilenames(
			path.Join(documentRoot, board.Dir, "thumb", upload.Filename))
	}
	if filter, err = gcsql.DoPostFiltering(post, upload, boardID, request, errEv, excludedFilterIDs...); err != nil {
		server.ServeError(writer, err.Error(), wantsJSON, nil)
		return
	}
	if HandleFilterAction(filter, post, upload, board, writer, request) {
		return
	}
	_, emailCommand := getEmailAndCommand(request)
	if err = post.Insert(emailCommand != "sage", board.ID, isLocked, isSticky, false, isCyclic); err != nil {
		errEv.Err(err).Caller().
			Str("sql", "postInsertion").
			Msg("Unable to insert post")
		if upload != nil {
			os.Remove(filePath)
			os.Remove(thumbPath)
			os.Remove(catalogThumbPath)
		}
		server.ServeError(writer, "Unable to insert post", wantsJSON, nil)
		return
	}

	if err = post.AttachFile(upload); err != nil {
		errEv.Err(err).Caller().
			Str("sql", "postInsertion").
			Msg("Unable to attach upload to post")
		os.Remove(filePath)
		os.Remove(thumbPath)
		os.Remove(catalogThumbPath)
		post.Delete()
		server.ServeError(writer, "Unable to attach upload", wantsJSON, map[string]any{
			"filename": upload.OriginalFilename,
		})
		return
	}
	if upload != nil {
		if err = config.TakeOwnership(filePath); err != nil {
			errEv.Err(err).Caller().
				Str("file", filePath).Send()
			os.Remove(filePath)
			os.Remove(thumbPath)
			os.Remove(catalogThumbPath)
			post.Delete()
			server.ServeError(writer, err.Error(), wantsJSON, nil)
		}
		if err = config.TakeOwnership(thumbPath); err != nil {
			errEv.Err(err).Caller().
				Str("thumbnail", thumbPath).Send()
			os.Remove(filePath)
			os.Remove(thumbPath)
			os.Remove(catalogThumbPath)
			post.Delete()
			server.ServeError(writer, err.Error(), wantsJSON, nil)
		}
		if err = config.TakeOwnership(catalogThumbPath); err != nil && !os.IsNotExist(err) {
			errEv.Err(err).Caller().
				Str("catalogThumbnail", catalogThumbPath).Send()
			os.Remove(filePath)
			os.Remove(thumbPath)
			os.Remove(catalogThumbPath)
			post.Delete()
			server.ServeError(writer, err.Error(), wantsJSON, nil)
		}
	}

	if !post.IsTopPost {
		toBePruned, err := post.CyclicPostsToBePruned()
		if err != nil {
			errEv.Err(err).Caller().Msg("Unable to get posts to be pruned from cyclic thread")
			server.ServeError(writer, "Unable to get cyclic thread info", wantsJSON, nil)
			return
		}
		gcutil.LogInt("toBePruned", len(toBePruned), infoEv, errEv)

		// prune posts from cyclic thread
		for _, prunePost := range toBePruned {
			fmt.Printf("%#v\n", prunePost)
			p := &gcsql.Post{ID: prunePost.PostID, ThreadID: prunePost.ThreadID}

			if err = p.Delete(); err != nil {
				errEv.Err(err).Caller().
					Int("postID", prunePost.PostID).
					Msg("Unable to prune post from cyclic thread")
				server.ServeError(writer, "Unable to prune post from cyclic thread", wantsJSON, nil)
				return
			}
			if prunePost.Filename != "" && prunePost.Filename != "deleted" {
				prunePostFile := path.Join(documentRoot, prunePost.Dir, "src", prunePost.Filename)
				prunePostThumbName, _ := uploads.GetThumbnailFilenames(prunePost.Filename)
				prunePostThumb := path.Join(documentRoot, prunePost.Dir, "thumb", prunePostThumbName)
				gcutil.LogStr("prunePostFile", prunePostFile, infoEv, errEv)
				gcutil.LogStr("prunePostThumb", prunePostThumb, infoEv, errEv)

				if err = os.Remove(prunePostFile); err != nil {
					errEv.Err(err).Caller().Msg("Unable to delete post file")
				}
				if err = os.Remove(prunePostThumb); err != nil {
					errEv.Err(err).Caller().Msg("Unable to delete post thumbnail")
				}
			}
		}
	}

	// rebuild the board page
	if err = building.BuildBoards(false, board.ID); err != nil {
		server.ServeError(writer, "Unable to build boards", wantsJSON, nil)
		return
	}

	if err = building.BuildFrontPage(); err != nil {
		server.ServeError(writer, "Unable to build front page", wantsJSON, nil)
		return
	}

	if wantsJSON {
		topPost, _ := post.TopPostID()
		server.ServeJSON(writer, map[string]any{
			"time":   post.CreatedOn,
			"id":     post.ID,
			"thread": config.WebPath(board.Dir, "/res/", strconv.Itoa(topPost)+".html"),
		})
		return
	}
	http.Redirect(writer, request, getRedirectURL(post, board, request), http.StatusFound)
}
