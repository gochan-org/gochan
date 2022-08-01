package posting

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

var (
	ErrInvalidReport   = errors.New("invalid report submitted")
	ErrInvalidPost     = errors.New("post does not exist")
	ErrNoReportedPosts = errors.New("no posts selected")
	ErrNoReportReason  = errors.New("no report reason given")
	ErrDuplicateReport = errors.New("post already reported")
)

func HandleReport(request *http.Request) error {
	board := request.FormValue("board")
	if request.Method != "POST" {
		return ErrInvalidReport
	}
	reportedPosts := []int{}
	var id int
	if !gcsql.DoesBoardExistByDir(board) {
		return gcsql.ErrBoardDoesNotExist
	}
	var err error
	for key, val := range request.Form {
		if _, err = fmt.Sscanf(key, "check%d", &id); err != nil || val[0] != "on" {
			err = nil
			continue
		}
		reportedPosts = append(reportedPosts, id)
	}
	if len(reportedPosts) == 0 {
		return ErrNoReportedPosts
	}
	ip := gcutil.GetRealIP(request)
	reason := strings.TrimSpace(request.PostFormValue("reason"))
	if reason == "" {
		return ErrNoReportReason
	}

	for _, postID := range reportedPosts {
		// check to see if the post has already been reported with this report string or if it can't be reported
		isDuplicate, isBlocked, err := gcsql.CheckPostReports(postID, reason)
		if err != nil {
			return err
		}
		if isDuplicate || isBlocked {
			// post has already been reported, and for the same reason, moving on
			continue
		}

		if _, err = gcsql.CreateReport(postID, ip, reason); err != nil {
			return err
		}
	}
	return nil
}
