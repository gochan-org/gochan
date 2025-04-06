package building

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"maps"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gctemplates"
	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting/uploads"
	"github.com/gochan-org/gochan/pkg/server/serverutil"
	"github.com/rs/zerolog"
)

var (
	bbcodeTagRE = regexp.MustCompile(`\[/?[^\[\]\s]+\]`)
)

type frontPagePost struct {
	Board         string
	URL           string
	ThumbURL      string
	FileDeleted   bool
	MessageSample string
	PostUploadBase
}

func (fp *frontPagePost) HasEmbed() bool {
	return strings.HasPrefix(fp.Filename, "embed:")
}

func (p *frontPagePost) GetEmbedThumbURL(board string) error {
	if !p.HasEmbed() {
		return nil
	}
	filenameParts := strings.SplitN(p.Filename, ":", 2)
	if len(filenameParts) != 2 {
		return fmt.Errorf("invalid embed filename: %s", p.Filename)
	}

	boardConfig := config.GetBoardConfig(board)

	_, thumbURLTmpl, err := boardConfig.GetEmbedTemplates(filenameParts[1])
	if err != nil {
		return err
	}
	if thumbURLTmpl == nil {
		return nil
	}

	templateData := config.EmbedTemplateData{
		MediaID: p.OriginalFilename,
	}
	var buf bytes.Buffer
	if err = thumbURLTmpl.Execute(&buf, templateData); err != nil {
		return err
	}
	p.ThumbURL = buf.String()

	return nil
}

func getFrontPagePosts(errEv *zerolog.Event) ([]frontPagePost, error) {
	siteCfg := config.GetSiteConfig()
	var query string

	if siteCfg.RecentPostsWithNoFile {
		// get recent posts, including those with no file
		query = "SELECT id, message_raw, dir, filename, original_filename, op_id FROM DBPREFIXv_front_page_posts"
	} else {
		query = "SELECT id, message_raw, dir, filename, original_filename, op_id FROM DBPREFIXv_front_page_posts_with_file"
	}
	query += " ORDER BY id DESC LIMIT " + strconv.Itoa(siteCfg.MaxRecentPosts)

	rows, cancel, err := gcsql.QueryTimeoutSQL(nil, query)
	defer cancel()
	if err != nil {
		errEv.Err(err).Caller().Send()
		return nil, err
	}
	defer rows.Close()
	var recentPosts []frontPagePost
	boardConfig := config.GetBoardConfig("")
	for rows.Next() {
		var post frontPagePost
		var id, topPostID string
		var message, boardDir, filename, originalFilename string
		err = rows.Scan(&id, &message, &boardDir, &filename, &originalFilename, &topPostID)
		if err != nil {
			errEv.Err(err).Caller().Send()
			return nil, err
		}
		message = bbcodeTagRE.ReplaceAllString(message, "")
		if len(message) > 40 {
			message = message[:37] + "..."
		}

		post = frontPagePost{
			Board:         boardDir,
			URL:           config.WebPath(boardDir, "res", topPostID+".html") + "#" + id,
			FileDeleted:   filename == "deleted",
			MessageSample: message,
			PostUploadBase: PostUploadBase{
				Filename:         filename,
				OriginalFilename: originalFilename,
				ThumbnailWidth:   boardConfig.ThumbWidthReply,
				ThumbnailHeight:  boardConfig.ThumbHeightReply,
			},
		}
		if !strings.HasPrefix(post.Filename, "embed:") {
			thumbnailFilename, _ := uploads.GetThumbnailFilenames(post.Filename)
			post.ThumbURL = config.WebPath(post.Board, "thumb", thumbnailFilename)
		}

		if post.HasEmbed() {
			if err = post.GetEmbedThumbURL(boardDir); err != nil {
				errEv.Err(err).Caller().Send()
				return nil, err
			}
			mediaID, _ := strings.CutPrefix(post.Filename, "embed:")
			_, thumbTmpl, _ := boardConfig.GetEmbedTemplates(mediaID)
			if thumbTmpl != nil {
				var buf bytes.Buffer
				if err = thumbTmpl.Execute(&buf, config.EmbedTemplateData{MediaID: post.OriginalFilename}); err != nil {
					errEv.Err(err).Caller().Send()
					return nil, err
				}
				post.ThumbURL = buf.String()
			}
		}
		recentPosts = append(recentPosts, post)
	}
	if err = rows.Close(); err != nil {
		errEv.Err(err).Caller().Send()
		return nil, err
	}
	return recentPosts, nil
}

// BuildFrontPage builds the front page using templates/front.html
func BuildFrontPage() error {
	errEv := gcutil.LogError(nil).
		Str("template", "front")
	defer errEv.Discard()
	err := gctemplates.InitTemplates(gctemplates.FrontPage)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("failed loading front page template: %w", err)
	}
	criticalCfg := config.GetSystemCriticalConfig()
	frontFile, err := os.OpenFile(path.Join(criticalCfg.DocumentRoot, "index.html"), os.O_CREATE|os.O_RDWR|os.O_TRUNC, config.NormalFileMode)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("failed opening front page for writing: %w", err)
	}

	if err = config.TakeOwnershipOfFile(frontFile); err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("failed setting file ownership for front page: %w", err)
	}

	var recentPostsArr []frontPagePost
	siteCfg := config.GetSiteConfig()
	recentPostsArr, err = getFrontPagePosts(errEv)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("failed loading recent posts: %w", err)
	}
	if err = serverutil.MinifyTemplate(gctemplates.FrontPage, map[string]any{
		"siteConfig":  siteCfg,
		"sections":    gcsql.AllSections,
		"boards":      gcsql.AllBoards,
		"boardConfig": config.GetBoardConfig(""),
		"recentPosts": recentPostsArr,
	}, frontFile, "text/html"); err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("failed executing front page template: %w", err)
	}
	return frontFile.Close()
}

// BuildPageHeader is a convenience function for automatically generating the top part
// of every normal HTML page
func BuildPageHeader(writer io.Writer, pageTitle string, board string, misc map[string]any) error {
	phMap := map[string]any{
		"pageTitle":     pageTitle,
		"documentTitle": pageTitle + " - " + config.GetSiteConfig().SiteName,
		"siteConfig":    config.GetSiteConfig(),
		"sections":      gcsql.AllSections,
		"boards":        gcsql.AllBoards,
		"boardConfig":   config.GetBoardConfig(board),
	}
	maps.Copy(phMap, misc)
	return serverutil.MinifyTemplate(gctemplates.PageHeader, phMap, writer, "text/html")
}

// BuildPageFooter is a convenience function for automatically generating the bottom
// of every normal HTML page
func BuildPageFooter(writer io.Writer) (err error) {
	return serverutil.MinifyTemplate(gctemplates.PageFooter,
		map[string]any{}, writer, "text/html")
}

// BuildJS minifies (if enabled) consts.js, which is built from a template
func BuildJS() error {
	// build consts.js from template
	err := gctemplates.InitTemplates(gctemplates.JsConsts)
	errEv := gcutil.LogError(nil).Str("building", "consts.js")
	defer errEv.Discard()
	if err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("failed loading consts.js template: %w", err)
	}

	boardCfg := config.GetBoardConfig("")
	criticalCfg := config.GetSystemCriticalConfig()
	constsJSPath := path.Join(criticalCfg.DocumentRoot, "js", "consts.js")
	constsJSFile, err := os.OpenFile(constsJSPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, config.NormalFileMode)
	if err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("failed opening consts.js for writing: %w", err)
	}

	if err = config.TakeOwnershipOfFile(constsJSFile); err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("unable to update file ownership for consts.js: %w", err)
	}

	if err = serverutil.MinifyTemplate(gctemplates.JsConsts, map[string]any{
		"styles":       boardCfg.Styles,
		"defaultStyle": boardCfg.DefaultStyle,
		"webroot":      criticalCfg.WebRoot,
		"timezone":     criticalCfg.TimeZone,
		"fileTypes":    boardCfg.AllowOtherExtensions,
	}, constsJSFile, "text/javascript"); err != nil {
		errEv.Err(err).Caller().Send()
		return fmt.Errorf("failed building consts.js: %w", err)
	}
	return constsJSFile.Close()
}

func embedMedia(post *Post) (template.HTML, error) {
	filenameParts := strings.SplitN(post.Filename, ":", 2)
	if len(filenameParts) != 2 {
		return "", errors.New("invalid embed ID")
	}

	boardCfg := config.GetBoardConfig(post.BoardDir)
	embedTmpl, thumbTmpl, err := boardCfg.GetEmbedTemplates(filenameParts[1])
	if err != nil {
		return "", err
	}

	templateData := config.EmbedTemplateData{
		MediaID:     post.OriginalFilename,
		HandlerID:   filenameParts[1],
		ThumbWidth:  boardCfg.ThumbWidth,
		ThumbHeight: boardCfg.ThumbHeight,
	}
	if !post.IsTopPost {
		templateData.ThumbWidth = boardCfg.ThumbWidthReply
		templateData.ThumbHeight = boardCfg.ThumbHeightReply
	}

	var buf bytes.Buffer
	if thumbTmpl != nil {
		if err := thumbTmpl.Execute(&buf, templateData); err != nil {
			return "", err
		}

		return template.HTML(fmt.Sprintf(
			`<img src=%q alt="Embedded video" class="embed thumb embed-%s" style="max-width: %dpx; max-height: %dpx;" embed-width="%d" embed-height="%d">`,
			buf.String(), filenameParts[1], templateData.ThumbWidth, templateData.ThumbHeight, boardCfg.EmbedWidth, boardCfg.EmbedHeight)), nil
	}

	if err = embedTmpl.Execute(&buf, templateData); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

func init() {
	gctemplates.AddTemplateFuncs(template.FuncMap{
		"embedMedia": embedMedia,
	})
}
