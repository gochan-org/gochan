package manage

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
)

type reportData struct {
	gcsql.Report
	StaffUser *string `json:"staff_user"`
	PostLink  string  `json:"post_link"`
}

func doReportHandling(request *http.Request, staff *gcsql.Staff, infoEv, errEv *zerolog.Event) error {
	doDismissAll := request.PostFormValue("dismiss-all")
	doDismissSel := request.PostFormValue("dismiss-sel")
	doBlockSel := request.PostFormValue("block-sel")

	if doDismissAll != "" {
		_, err := gcsql.Exec(nil, `UPDATE DBPREFIXreports SET is_cleared = 1`)
		if err != nil {
			errEv.Err(err).Caller().Send()
			return err
		}
		infoEv.Msg("All reports dismissed")
		return nil
	}

	if doDismissSel == "" && doBlockSel == "" {
		return nil
	}

	if doBlockSel != "" && staff.Rank != 3 {
		gcutil.LogWarning().Caller().
			Str("IP", gcutil.GetRealIP(request)).
			Str("staff", staff.Username).
			Str("rejected", "not an admin").
			Msg("only the administrator can block reports")
		return server.NewServerError("only the administrator can block reports", http.StatusForbidden)
	}

	var checkedReports []int
	for reportIDstr, val := range request.PostForm {
		if len(val) == 0 {
			continue
		}

		idStr, ok := strings.CutPrefix(reportIDstr, "report")
		if !ok {
			continue
		}
		id, err := strconv.Atoi(idStr)
		if err != nil {
			continue
		}
		checkedReports = append(checkedReports, id)
	}

	if len(checkedReports) == 0 {
		return nil
	}
	gcutil.LogArray("reportIDs", checkedReports, infoEv)

	for _, reportID := range checkedReports {
		matched, err := gcsql.ClearReport(reportID, staff.ID, doBlockSel != "")
		if !matched {
			errEv.Err(err).Caller().
				Int("reportID", reportID).
				Msg("report not found")
			return server.NewServerError(fmt.Sprintf("report with id %d does not exist or is cleared", reportID), http.StatusBadRequest)
		}
		if err != nil {
			errEv.Err(err).Caller().
				Int("reportID", reportID).
				Msg("failed to clear report")
			return server.NewServerError(fmt.Sprintf("failed to clear report with id %d", reportID), http.StatusInternalServerError)
		}
	}
	infoEv.Msg("Reports dismissed")
	return nil
}

func reportsCallback(_ http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, infoEv *zerolog.Event, errEv *zerolog.Event) (output any, err error) {
	if err = doReportHandling(request, staff, infoEv, errEv); err != nil {
		errEv.Discard() // doReportHandling logs errors
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.DefaultSQLTimeout*time.Second)
	defer cancel()

	requestOptions := &gcsql.RequestOptions{
		Context: ctx,
		Cancel:  cancel,
	}

	if err = gcsql.DeleteReportsOfDeletedPosts(requestOptions); err != nil {
		errEv.Err(err).Caller().Send()
		return nil, server.NewServerError("failed to clean up reports of deleted posts", http.StatusInternalServerError)
	}

	rows, err := gcsql.Query(requestOptions, `SELECT id, staff_id, staff_user, post_id, ip, reason, is_cleared FROM DBPREFIXv_post_reports`)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return nil, err
	}
	defer rows.Close()
	// reports := make([]map[string]any, 0)
	var reports []reportData
	for rows.Next() {
		var report reportData
		err = rows.Scan(&report.ID, &report.HandledByStaffID, &report.StaffUser, &report.PostID, &report.IP, &report.Reason, &report.IsCleared)
		if report.StaffUser == nil {
			user := "unassigned"
			report.StaffUser = &user
			handledByStaffID := 0
			report.HandledByStaffID = &handledByStaffID
		}
		if err != nil {
			errEv.Err(err).Caller().Send()
			return nil, server.NewServerError("failed to scan report row", http.StatusInternalServerError)
		}

		post, err := gcsql.GetPostFromID(report.PostID, true, requestOptions)
		if err != nil {
			errEv.Err(err).Caller().Msg("failed to get post from ID")
			return nil, server.NewServerError("failed to get post from ID", http.StatusInternalServerError)
		}
		report.PostLink = post.WebPath()
		reports = append(reports, report)
	}
	if err = rows.Close(); err != nil {
		errEv.Err(err).Caller().Send()
		return nil, err
	}
	if wantsJSON {
		return reports, nil
	}

	reportsBuffer := bytes.NewBufferString("")
	err = serverutil.MinifyTemplate(gctemplates.ManageReports,
		map[string]any{
			"reports": reports,
			"staff":   staff,
		}, reportsBuffer, "text/html")
	if err != nil {
		errEv.Err(err).Caller().Send()
		return "", err
	}
	output = reportsBuffer.String()
	return
}
