package manage

import (
	"errors"
	"html"
	"net/http"
	"strconv"
	"time"

	"github.com/Eggbertx/durationutil"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
)

func ipBanFromRequest(ban *gcsql.IPBan, request *http.Request, infoEv *zerolog.Event, errEv *zerolog.Event) error {
	now := time.Now()
	editBanIDStr := request.FormValue("edit")
	if editBanIDStr != "" && request.PostFormValue("do") == "edit" {
		banID, err := strconv.Atoi(editBanIDStr)
		if err != nil {
			errEv.Err(err).Caller().
				Str("editBanID", editBanIDStr).Send()
			return errors.New("invalid 'edit' field value (must be int)")
		}
		editing, err := gcsql.GetIPBanByID(banID)
		if err != nil {
			errEv.Err(err).Caller().
				Int("editBanID", banID).Send()
			return errors.New("Unable to get ban with id " + editBanIDStr + " (SQL error)")
		}
		*ban = *editing
		return nil
	}
	var err error
	ip := request.PostFormValue("ip")
	ban.RangeStart, ban.RangeEnd, err = gcutil.ParseIPRange(ip)
	if err != nil {
		errEv.Err(err).Caller().
			Str("ip", ip)
	}
	gcutil.LogStr("rangeStart", ban.RangeStart, infoEv, errEv)
	gcutil.LogStr("rangeEnd", ban.RangeEnd, infoEv, errEv)

	durationStr := request.PostFormValue("duration")
	ban.Permanent = durationStr == ""
	if !ban.Permanent {
		duration, err := durationutil.ParseLongerDuration(durationStr)
		if err != nil {
			errEv.Err(err).Caller().
				Str("duration", durationStr).
				Msg("Invalid duration")
			return err
		}
		ban.ExpiresAt = now.Add(duration)
	}

	ban.CanAppeal = request.PostFormValue("noappeals") != "on"
	if ban.CanAppeal {
		appealWaitStr := request.PostFormValue("appealwait")
		if appealWaitStr != "" {
			appealDuration, err := durationutil.ParseLongerDuration(appealWaitStr)
			if err != nil {
				errEv.Err(err).Caller().
					Str("appealwait", appealWaitStr).
					Msg("Invalid appeal delay duration string")
				return err
			}
			ban.AppealAt = now.Add(appealDuration)
		} else {
			ban.AppealAt = now
		}
	} else {
		ban.AppealAt = now
	}

	ban.IsThreadBan = request.PostFormValue("threadban") == "on"
	boardIDstr := request.PostFormValue("boardid")
	if boardIDstr != "" && boardIDstr != "0" {
		boardID, err := strconv.Atoi(boardIDstr)
		if err != nil {
			errEv.Err(err).Caller().
				Str("boardid", boardIDstr).Send()
			return err
		}
		gcutil.LogInt("boardID", boardID, infoEv, errEv)
		ban.BoardID = new(int)
		*ban.BoardID = boardID
	}
	ban.Message = html.EscapeString(request.PostFormValue("reason"))
	ban.StaffNote = html.EscapeString(request.PostFormValue("staffnote"))
	ban.IsActive = true
	gcutil.LogStr("banMessage", request.PostFormValue("reason"), infoEv, errEv)
	gcutil.LogStr("staffNote", request.PostFormValue("staffnote"), infoEv, errEv)
	return gcsql.NewIPBan(ban)
}
