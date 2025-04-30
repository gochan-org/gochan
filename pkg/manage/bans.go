package manage

import (
	"fmt"
	"html"
	"time"

	"github.com/Eggbertx/durationutil"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/rs/zerolog"
)

type banPageFields struct {
	Do               string `form:"do" method:"POST"`
	IP               string `form:"ip,requred,notempty" method:"POST"`
	Duration         string `form:"duration" method:"POST"`
	AppealWaitTime   string `form:"appealwait" method:"POST"`
	NoAppeals        bool   `form:"noappeals" method:"POST"`
	ThreadBan        bool   `form:"threadban" method:"POST"`
	BoardID          int    `form:"boardid" method:"POST"`
	PostID           int    `form:"postid"`
	UseBannedMessage bool   `form:"usebannedmessage" method:"POST"`
	BannedMessage    string `form:"bannedmessage" method:"POST"`
	Reason           string `form:"reason" method:"POST"`
	StaffNote        string `form:"staffnote" method:"POST"`

	FilterBoardID int `form:"filterboardid"`
	Limit         int `form:"limit,default=200"`
	DeleteID      int `form:"delete"`
}

func (bpf *banPageFields) fillBanFields(ban *gcsql.IPBan, infoEv, errEv *zerolog.Event) (err error) {
	ban.RangeStart, ban.RangeEnd, err = gcutil.ParseIPRange(bpf.IP)
	if err != nil {
		errEv.Err(err).Caller().
			Str("ip", bpf.IP).Send()
		return fmt.Errorf("unable to parse IP range: %w", err)
	}
	if ban.RangeStart == ban.RangeEnd {
		gcutil.LogStr("ip", bpf.IP, infoEv, errEv)
	} else {
		gcutil.LogStr("rangeStart", ban.RangeStart, infoEv, errEv)
		gcutil.LogStr("rangeEnd", ban.RangeEnd, infoEv, errEv)
	}
	ban.Permanent = bpf.Duration == ""
	if !ban.Permanent {
		duration, err := durationutil.ParseLongerDuration(bpf.Duration)
		if err != nil {
			errEv.Err(err).Caller().
				Str("duration", bpf.Duration).
				Msg("Invalid duration")
			return err
		}
		ban.ExpiresAt = time.Now().Add(duration)
	}
	ban.CanAppeal = !bpf.NoAppeals
	gcutil.LogBool("appealable", ban.CanAppeal, infoEv, errEv)
	if ban.CanAppeal {
		if bpf.AppealWaitTime != "" {
			duration, err := durationutil.ParseLongerDuration(bpf.AppealWaitTime)
			if err != nil {
				errEv.Err(err).Caller().
					Str("appealwait", bpf.AppealWaitTime).
					Msg("Invalid appeal delay duration string")
				return err
			}
			ban.AppealAt = time.Now().Add(duration)
		} else {
			ban.AppealAt = time.Now()
		}
	} else {
		ban.AppealAt = time.Now()
	}
	ban.IsThreadBan = bpf.ThreadBan
	if bpf.BoardID != 0 {
		ban.BoardID = &bpf.BoardID
	}
	ban.Message = html.EscapeString(bpf.Reason)
	ban.StaffNote = html.EscapeString(bpf.StaffNote)
	ban.IsActive = true
	gcutil.LogStr("reason", ban.Message, infoEv, errEv)
	return nil
}
