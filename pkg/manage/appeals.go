package manage

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/Eggbertx/go-forms"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/server"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
	"github.com/uptrace/bunrouter"
)

func appealsCallback(_ http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, logger zerolog.Logger) (output any, err error) {
	var form appealsForm
	if err = forms.FillStructFromForm(request, &form); err != nil {
		logger.Err(err).Caller().
			Msg("Unable to fill struct from form")
		return "", server.NewServerError(err, http.StatusBadRequest)
	}

	if request.Method == http.MethodPost {
		if err = form.validate(); err != nil {
			logger.Err(err).Caller().
				Msg("Invalid form data")
			return "", err
		}

		zlArr := zerolog.Arr()
		for _, v := range form.AppealIDs {
			zlArr.Int(v)
		}
		if form.isApprove() {
			logger = logger.With().Array("approveAppeals", zlArr).Logger()
		} else if form.isDeny() {
			logger = logger.With().Array("denyAppeals", zlArr).Logger()
		}
	} else {
		if form.Limit < 1 {
			form.Limit = 20
		}
	}

	if form.isApprove() {
		for _, approveID := range form.AppealIDs {
			if err = gcsql.ApproveAppeal(approveID, staff.ID); err != nil {
				logger.Err(err).Caller().
					Int("approveAppeal", approveID).Send()
				return "", err
			}
		}
		logger.Info().Msg("Approved appeal(s)")
	} else if form.isDeny() {
		return "", server.NewServerError("deny appeal not yet implemented", http.StatusNotImplemented)
	}

	appeals, err := gcsql.GetAppeals(gcsql.AppealsQueryOptions{
		Limit:           form.Limit,
		Active:          gcsql.OnlyTrue,
		Unexpired:       gcsql.OnlyTrue,
		OrderDescending: true,
	})
	if err != nil {
		logger.Err(err).Caller().Send()
		return "", fmt.Errorf("failed to get appeals list: %w", err)
	}

	if wantsJSON {
		return appeals, nil
	}
	var buf bytes.Buffer
	pageData := map[string]any{}
	if len(appeals) > 0 {
		pageData["appeals"] = appeals
	}
	if err = serverutil.MinifyTemplate(gctemplates.ManageAppeals, pageData, &buf, "text/html"); err != nil {
		logger.Err(err).Str("template", gctemplates.ManageAppeals).Caller().Send()
		return "", fmt.Errorf("failed executing appeal management page template: %w", err)
	}
	return buf.String(), err
}

func appealConversationCallback(_ http.ResponseWriter, request *http.Request, staff *gcsql.Staff, wantsJSON bool, logger zerolog.Logger) (output any, err error) {
	params, _ := request.Context().Value(requestContextKey{}).(bunrouter.Params)
	appealID, err := params.Int("appealID")
	if err != nil {
		logger.Err(err).Caller().Msg("Appeal ID is not a valid integer")
		return nil, server.NewServerError("missing appeal ID", http.StatusBadRequest)
	}

	data := map[string]any{
		"appealID": appealID,
	}

	SetCustomPageTitle(request, fmt.Sprintf("Appeal %d Conversation", appealID))

	var buf bytes.Buffer
	if err = serverutil.MinifyTemplate(gctemplates.ManageAppealConversation, data, &buf, "text/html"); err != nil {
		logger.Err(err).Str("template", gctemplates.ManageAppealConversation).Caller().Send()
		return "", fmt.Errorf("failed executing appeal conversation page template: %w", err)
	}
	return buf.String(), err

}
