package manage

import (
	"errors"
	"html"
	"net/http"
	"strconv"
	"time"

	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
)

func ipBanFromRequest(ban *gcsql.IPBan, request *http.Request, errEv *zerolog.Event) error {
	banIDStr := request.FormValue("edit")
	if banIDStr != "" && request.FormValue("do") == "edit" {
		banID, err := strconv.Atoi(banIDStr)
		if err != nil {
			errEv.Err(err).
				Str("editBanID", banIDStr).
				Caller().Send()
			return errors.New("invalid 'edit' field value (must be int)")
		}
		editing, err := gcsql.GetIPBanByID(banID)
		if err != nil {
			errEv.Err(err).
				Int("editBanID", banID).
				Caller().Send()
			return errors.New("Unable to get ban with id " + banIDStr + " (SQL error)")
		}
		*ban = *editing
		return nil
	}
	ban.IP = request.FormValue("ip")
	ban.Permanent = request.FormValue("permanent") == "on"
	if !ban.Permanent {
		durationStr := request.FormValue("duration")
		duration, err := gcutil.ParseDurationString(durationStr)
		if err != nil {
			errEv.Err(err).
				Str("duration", durationStr).
				Caller().Msg("Invalid duration")
			return err
		}
		ban.ExpiresAt = time.Now().Add(duration)
	}

	ban.CanAppeal = request.FormValue("noappeals") != "on"
	if ban.CanAppeal {
		appealWaitStr := request.FormValue("appealwait")
		if appealWaitStr != "" {
			appealDuration, err := gcutil.ParseDurationString(appealWaitStr)
			if err != nil {
				errEv.Err(err).
					Str("appealwait", appealWaitStr).
					Caller().Msg("Invalid appeal delay duration string")
				return err
			}
			ban.AppealAt = time.Now().Add(appealDuration)
		}
	}

	ban.IsThreadBan = request.FormValue("threadban") == "on"
	boardIDstr := request.FormValue("boardid")
	if boardIDstr != "" && boardIDstr != "0" {
		boardID, err := strconv.Atoi(boardIDstr)
		if err != nil {
			errEv.Err(err).
				Str("boardid", boardIDstr).
				Caller().Send()
			return err
		}
		ban.BoardID = new(int)
		*ban.BoardID = boardID
	}
	ban.Message = html.EscapeString(request.FormValue("reason"))
	ban.StaffNote = html.EscapeString(request.FormValue("staffnote"))
	ban.IsActive = true
	return gcsql.NewIPBan(ban)
}
