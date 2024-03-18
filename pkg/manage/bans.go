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
	banIDStr := request.FormValue("edit")
	if banIDStr != "" && request.FormValue("do") == "edit" {
		banID, err := strconv.Atoi(banIDStr)
		if err != nil {
			errEv.Err(err).Caller().
				Str("editBanID", banIDStr).Send()
			return errors.New("invalid 'edit' field value (must be int)")
		}
		editing, err := gcsql.GetIPBanByID(banID)
		if err != nil {
			errEv.Err(err).Caller().
				Int("editBanID", banID).Send()
			return errors.New("Unable to get ban with id " + banIDStr + " (SQL error)")
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

	ban.Permanent = request.FormValue("permanent") == "on"
	if ban.Permanent {
		ban.ExpiresAt = now
	} else {
		durationStr := request.FormValue("duration")
		duration, err := durationutil.ParseLongerDuration(durationStr)
		if err != nil {
			errEv.Err(err).Caller().
				Str("duration", durationStr).
				Msg("Invalid duration")
			return err
		}
		ban.ExpiresAt = now.Add(duration)
	}

	ban.CanAppeal = request.FormValue("noappeals") != "on"
	if ban.CanAppeal {
		appealWaitStr := request.FormValue("appealwait")
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

	ban.IsThreadBan = request.FormValue("threadban") == "on"
	boardIDstr := request.FormValue("boardid")
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
	ban.Message = html.EscapeString(request.FormValue("reason"))
	ban.StaffNote = html.EscapeString(request.FormValue("staffnote"))
	ban.IsActive = true
	gcutil.LogStr("banMessage", request.FormValue("reason"), infoEv, errEv)
	gcutil.LogStr("staffNote", request.FormValue("staffnote"), infoEv, errEv)
	return gcsql.NewIPBan(ban)
}
