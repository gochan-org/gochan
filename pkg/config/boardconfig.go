package config

import (
	"encoding/json"
	"errors"
	"html/template"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/gochan-org/gochan/pkg/gcutil"
	"github.com/gochan-org/gochan/pkg/posting/geoip"
)

// BoardConfig contains information about a specific board to be stored in /path/to/board/board.json
// or all boards if it is stored in the main gochan.json file. If a board doesn't have board.json,
// the site's default board config (with values set in gochan.json) will be used
type BoardConfig struct {
	// MaxThreads is the number of threads that will be kept in the boards directory, before pruning old ones. If set to 0, pruning is disabled.
	// This also determines the number of pages that will be kept.
	// Default: 200
	MaxThreads int

	// ThreadsPerPage is the number of threads to display per page
	// Default: 20
	ThreadsPerPage int

	// InheritGlobalStyles determines whether to use the global styles in addition to the board's styles, as opposed to only the board's styles
	// Default: true
	InheritGlobalStyles bool

	// Styles is a list of Gochan themes with Name and Filename fields, choosable by the user
	Styles []Style

	// DefaultStyle is the filename of the default style to use for the board or the site. If it is not set, the first style in the Styles list will be used
	// Default: pipes.css
	DefaultStyle string

	// IncludeGlobalStyles is a list of additional CSS files to be loaded on the board pages, or all pages if this is the global configuration.
	IncludeGlobalStyles []string

	// IncludeScripts is a list of additional scripts to be loaded on the board pages, or all pages if this is the global configuration.
	IncludeScripts []IncludeScript

	// Banners is a list of banners to display on the board's front page, with Filename, Width, and Height fields
	Banners []PageBanner

	// Lockdown prevents users from posting if true
	Lockdown bool

	// LockdownMessage is the message displayed to users if they try to cretae a post when the site is in lockdown
	// Default: This imageboard has temporarily disabled posting. We apologize for the inconvenience
	LockdownMessage string

	PostConfig
	UploadConfig

	// DateTimeFormat is the human readable format to use for showing post timestamps. See [the official documentation](https://pkg.go.dev/time#Time.Format) for more information.
	// Default: Mon, January 02, 2006 3:04:05 PM
	DateTimeFormat string

	// ShowPosterID determines whether to show the generated thread-unique poster ID in the post header (not yet implemented)
	ShowPosterID bool

	// EnableSpoileredImages determines whether to allow users to spoiler images (not yet implemented)
	// Default: true
	EnableSpoileredImages bool

	// EnableSpoileredThreads determines whether to allow users to spoiler threads (not yet implemented)
	// Default: true
	EnableSpoileredThreads bool

	// Worksafe determines whether the board is worksafe or not. If it is set to true, threads cannot be marked NSFW
	// (given a hashtag with the text NSFW, case insensitive).
	// Default: true
	Worksafe bool

	// Cooldowns is used to prevent spamming by setting the number of seconds the user must wait before creating new threads or replies
	// Default: See BoardCooldowns section
	Cooldowns BoardCooldowns

	// RenderURLsAsLinks determines whether to render URLs as clickable links in posts
	// Default: true
	RenderURLsAsLinks bool

	// EnableCatalog determines whether to build a catalog page for the board (or all boards if this is the global configuration).
	// Default: true
	EnableCatalog bool

	// EnableGeoIP shows a dropdown box allowing the user to set their post flag as their country
	EnableGeoIP bool

	// EnableNoFlag allows the user to post without a flag. It is only used if EnableGeoIP or CustomFlags is true
	EnableNoFlag bool

	// CustomFlags is a list of non-geoip flags with Name (viewable to the user) and Flag (flag image filename) fields. See geoip.Country section for more information.
	CustomFlags []geoip.Country

	isGlobal        bool
	boardConfigPath string
}

// CheckCustomFlag returns true if the given flag and name are configured for
// the board (or are globally set)
func (bc *BoardConfig) CheckCustomFlag(flag string) (string, bool) {
	for _, country := range bc.CustomFlags {
		if flag == country.Flag {
			return country.Name, true
		}
	}
	return "", false
}

func (bc *BoardConfig) validateBoardConfig() error {
	if bc.MaxThreads <= 0 {
		bc.MaxThreads = defaultGochanConfig.MaxThreads
	}
	if bc.ThreadsPerPage <= 0 {
		bc.ThreadsPerPage = defaultGochanConfig.ThreadsPerPage
	}
	if bc.isGlobal && bc.DefaultStyle == "" {
		bc.DefaultStyle = defaultGochanConfig.DefaultStyle
	}
	if bc.isGlobal && bc.LockdownMessage == "" {
		bc.LockdownMessage = defaultGochanConfig.LockdownMessage
	}
	if bc.isGlobal && bc.DateTimeFormat == "" {
		bc.DateTimeFormat = defaultGochanConfig.DateTimeFormat
	}
	if bc.Cooldowns.NewThread <= 0 {
		bc.Cooldowns.NewThread = defaultGochanConfig.Cooldowns.NewThread
	}
	if bc.Cooldowns.Reply <= 0 {
		bc.Cooldowns.Reply = defaultGochanConfig.Cooldowns.Reply
	}
	if bc.Cooldowns.ImageReply <= 0 {
		bc.Cooldowns.ImageReply = defaultGochanConfig.Cooldowns.ImageReply
	}
	if bc.AnonymousName == "" {
		bc.AnonymousName = defaultGochanConfig.AnonymousName
	}
	if bc.AutosageAfter <= 0 {
		bc.AutosageAfter = defaultGochanConfig.AutosageAfter
	}
	if bc.NoUploadsAfter <= 0 {
		bc.NoUploadsAfter = defaultGochanConfig.NoUploadsAfter
	}
	if bc.MaxMessageLength <= 0 {
		bc.MaxMessageLength = defaultGochanConfig.MaxMessageLength
	}
	if bc.MinMessageLength < 0 {
		bc.MinMessageLength = 0
	}
	if bc.MinMessageLength > bc.MaxMessageLength {
		return &InvalidValueError{
			Field:   "MinMessageLength",
			Value:   bc.MinMessageLength,
			Details: "MinMessageLength cannot be greater than MaxMessageLength",
		}
	}
	if bc.RepliesOnBoardPage <= 0 {
		bc.RepliesOnBoardPage = defaultGochanConfig.RepliesOnBoardPage
	}
	if bc.StickyRepliesOnBoardPage < 0 {
		bc.StickyRepliesOnBoardPage = defaultGochanConfig.StickyRepliesOnBoardPage
	}
	if bc.CyclicThreadNumPosts <= 0 {
		bc.CyclicThreadNumPosts = defaultGochanConfig.CyclicThreadNumPosts
	}
	if bc.BanMessage == "" {
		bc.BanMessage = defaultGochanConfig.BanMessage
	}
	if bc.EmbedWidth <= 0 {
		bc.EmbedWidth = defaultGochanConfig.EmbedWidth
	}
	if bc.EmbedHeight <= 0 {
		bc.EmbedHeight = defaultGochanConfig.EmbedHeight
	}
	if bc.MaxFileSize <= 0 {
		bc.MaxFileSize = defaultGochanConfig.MaxFileSize
	}
	if bc.ThumbWidth <= 0 {
		bc.ThumbWidth = defaultGochanConfig.ThumbWidth
	}
	if bc.ThumbHeight <= 0 {
		bc.ThumbHeight = defaultGochanConfig.ThumbHeight
	}
	if bc.ThumbWidthReply <= 0 {
		bc.ThumbWidthReply = defaultGochanConfig.ThumbWidthReply
	}
	if bc.ThumbHeightReply <= 0 {
		bc.ThumbHeightReply = defaultGochanConfig.ThumbHeightReply
	}
	if bc.ThumbWidthCatalog <= 0 {
		bc.ThumbWidthCatalog = defaultGochanConfig.ThumbWidthCatalog
	}
	if bc.ThumbHeightCatalog <= 0 {
		bc.ThumbHeightCatalog = defaultGochanConfig.ThumbHeightCatalog
	}

	return bc.validateEmbedMatchers()
}

// IsGlobal returns true if this is the global configuration applied to all
// boards by default, or false if it is an explicitly configured board
func (bc *BoardConfig) IsGlobal() bool {
	return bc.isGlobal
}

type BoardCooldowns struct {
	// NewThread is the number of seconds the user must wait before creating new threads.
	// Default: 30
	NewThread int `json:"threads"`

	// NewReply is the number of seconds the user must wait after replying to a thread before they can create another reply.
	// Default: 7
	Reply int `json:"replies"`

	// NewImageReply is the number of seconds the user must wait after replying to a thread with an upload before they can create another reply.
	// Default: 7
	ImageReply int `json:"images"`
}

// PageBanner represents the filename and dimensions of a banner image to display on board and thread pages
type PageBanner struct {
	// Filename is the name of the image file to display as seen by the browser
	Filename string
	// Width is the width of the image in pixels
	Width int `json:",omitempty"`
	// Height is the height of the image in pixels
	Height int `json:",omitempty"`
}

// Style represents a theme (Pipes, Dark, etc)
type Style struct {
	// Name is the display name of the style
	Name string

	// Filename is the name of the CSS file in /static to use for the style
	Filename string
}

type IncludeScript struct {
	// Location is the path or URL to the script to include
	Location string

	// Defer tells the browser to load the script after the page has loaded if it is true
	Defer bool
}

type PostConfig struct {
	// AnonymousName is the name used for anonymous posts when the name field is left empty
	// Default: Anonymous
	AnonymousName string

	// ForceAnonymous determines whether to force all posts to be anonymous
	ForceAnonymous bool

	// AutosageAfter is the number of replies after which a thread will no longer be bumped when a new post is made
	// Default: 200
	AutosageAfter int

	// NoUploadsAfter is the number of uploads (or embeds) in a thread after which uploads will no longer be allowed in that thread.
	// If < 0, then uploads are allowed indefinitely.
	// Default: -1
	NoUploadsAfter int

	// MinMessageLength is the minimum number of characters required in a post
	MinMessageLength int

	// MaxMessageLength is the maximum number of characters allowed in a post
	// Default: 2000
	MaxMessageLength int

	// ReservedTrips is used for reserving secure tripcodes. It should be a map of input strings to output tripcode strings. For example, if you have `{"abcd":"WXYZ"}` and someone posts with the name Name##abcd, their name will instead show up as Name!!WXYZ on the site.
	ReservedTrips map[string]string

	// RepliesOnBoardPage is the number of replies to display on the board page
	// Default: 3
	RepliesOnBoardPage int

	// StickyRepliesOnBoardPage is the number of replies to display on the board page for sticky threads
	// Default: 1
	StickyRepliesOnBoardPage int

	// NewThreadsRequireUpload determines whether to require an upload to create a new thread
	NewThreadsRequireUpload bool

	// AllPostsRequireUpload determines whether to require an upload for all posts, both OP and replies
	AllPostsRequireUpload bool

	// EnableCyclicThreads allows users to create threads that have a maximum number of replies before the oldest reply is deleted
	// Default: true
	EnableCyclicThreads bool

	// CyclicThreadNumPost determines the number of posts a cyclic thread can have before the oldest post is deleted
	// Default: 500
	CyclicThreadNumPosts int

	// BanColors is a list of colors to use for the ban message with the staff name as the key. If the staff name is not found in the list, the default style color will be used.
	BanColors map[string]string

	// BanMessage is the default message shown on a post that a user was banned for
	// Default: USER WAS BANNED FOR THIS POST
	BanMessage string

	// AllowEmbeds determines whether to allow embedding external media files in posts
	AllowEmbeds bool

	// EmbedWidth is the width of embedded external media files
	// Default: 400
	EmbedWidth int

	// EmbedHeight is the height of embedded external media files
	// Default: 300
	EmbedHeight int

	// EmbedMatchers is a map of site ID keys to objects used to match (via regular expression) URLs and embed them in posts via templates,
	// with an optional image thumbnail if supported. If a URL template is not provided, the video/frame will be embedded directly.
	// If EmbedMatchers is nil, embedding is disabled for the board, or globally if it is in the global configuration.
	EmbedMatchers                     map[string]EmbedMatcher
	embedMatchersRegex                map[string]*regexp.Regexp
	embedMatchersEmbedTemplate        map[string]*template.Template
	embedMatchersThumbnailURLTemplate map[string]*template.Template
	embedMatchersMediaURLTemplate     map[string]*template.Template

	// ImagesOpenNewTab determines whether to open images in a new tab when an image link is clicked
	// Default: true
	ImagesOpenNewTab bool

	// NewTabOnExternalLinks determines whether to open external links in a new tab
	// Default: true
	NewTabOnExternalLinks bool

	// NewTabOnOutlinks is an alias for the NewTabOnExternalLinks field.
	//
	// Deprecated: Use NewTabOnExternalLinks instead
	NewTabOnOutlinks bool `json:",omitempty"`

	// EnableBBcode will render BBCode tags to HTML if true
	// Default: true
	EnableBBcode bool

	// AllowDiceRerolls determines whether to allow users to edit posts to reroll dice
	AllowDiceRerolls bool

	RedirectToThread bool
}

// HasEmbedMatchers returns true if the board has embed handlers configured
func (pc *PostConfig) HasEmbedMatchers() bool {
	return len(pc.EmbedMatchers) > 0
}

// GetEmbedMediaID returns the site ID, and media ID for the given URL if it is compatible with any
// configured embed handlers. It returns an error if none are found
func (pc *PostConfig) GetEmbedMediaID(url string) (string, string, error) {
	if pc.embedMatchersRegex == nil {
		pc.embedMatchersRegex = make(map[string]*regexp.Regexp)
	}
	var err error
	for m, matcher := range pc.EmbedMatchers {
		re, ok := pc.embedMatchersRegex[m]
		if !ok {
			re, err = regexp.Compile(matcher.URLRegex)
			if err != nil {
				return "", "", err
			}
		}
		matches := re.FindAllStringSubmatch(url, -1)
		if len(matches) == 1 {
			pc.embedMatchersRegex[m] = re
			submatchIndex := 1
			if matcher.MediaIDSubmatchIndex != nil {
				submatchIndex = *matcher.MediaIDSubmatchIndex
			}
			return m, matches[0][submatchIndex], nil
		}
	}
	return "", "", ErrNoMatchingEmbedHandler
}

// GetEmbedTemplates returns the embed and (if it has one) thumbnail URL templates for the given embed ID
func (pc *PostConfig) GetEmbedTemplates(embedID string) (*template.Template, *template.Template, error) {
	matcher, ok := pc.EmbedMatchers[embedID]
	if !ok {
		return nil, nil, ErrNoMatchingEmbedHandler
	}
	embedTmpl, ok := pc.embedMatchersEmbedTemplate[embedID]
	var err error
	if !ok {
		pc.embedMatchersEmbedTemplate[embedID], err = template.New(embedID + "frame").Parse(matcher.EmbedTemplate)
		if err != nil {
			return nil, nil, err
		}
		embedTmpl = pc.embedMatchersEmbedTemplate[embedID]
	}
	thumbTmpl, ok := pc.embedMatchersThumbnailURLTemplate[embedID]
	if !ok {
		if matcher.ThumbnailURLTemplate != "" {
			pc.embedMatchersThumbnailURLTemplate[embedID], err = template.New(embedID + "thumb").Parse(matcher.ThumbnailURLTemplate)
			if err != nil {
				return nil, nil, err
			}
			thumbTmpl = pc.embedMatchersThumbnailURLTemplate[embedID]
		} else {
			pc.embedMatchersThumbnailURLTemplate[embedID] = nil
		}
	}
	return embedTmpl, thumbTmpl, nil
}

func (pc *PostConfig) GetLinkTemplate(embedID string) (*template.Template, error) {
	_, ok := pc.embedMatchersMediaURLTemplate[embedID]
	if !ok {
		matcher, ok := pc.EmbedMatchers[embedID]
		if !ok {
			return nil, ErrNoMatchingEmbedHandler
		}
		var err error
		pc.embedMatchersMediaURLTemplate[embedID], err = template.New(embedID + "url").Parse(matcher.MediaURLTemplate)
		return nil, err
	}
	return pc.embedMatchersMediaURLTemplate[embedID], nil
}

func (pc *PostConfig) validateEmbedMatchers() error {
	if pc.EmbedMatchers == nil {
		return nil
	}
	if pc.embedMatchersRegex == nil {
		pc.embedMatchersRegex = map[string]*regexp.Regexp{}
	}
	if pc.embedMatchersEmbedTemplate == nil {
		pc.embedMatchersEmbedTemplate = map[string]*template.Template{}
	}
	if pc.embedMatchersThumbnailURLTemplate == nil {
		pc.embedMatchersThumbnailURLTemplate = map[string]*template.Template{}
	}
	if pc.embedMatchersMediaURLTemplate == nil {
		pc.embedMatchersMediaURLTemplate = map[string]*template.Template{}
	}

	for m, matcher := range pc.EmbedMatchers {
		if _, exists := pc.embedMatchersRegex[m]; exists {
			// already registered and validated
			continue
		}
		re, err := regexp.Compile(matcher.URLRegex)
		if err != nil {
			return &InvalidValueError{
				Field:   "EmbedMatchers[" + m + "].URLRegex",
				Value:   matcher.URLRegex,
				Details: err.Error(),
			}
		}
		pc.embedMatchersRegex[m] = re
		tmpl, err := template.New(m + "frame").Parse(matcher.EmbedTemplate)
		if err != nil {
			return &InvalidValueError{
				Field:   "EmbedMatchers[" + m + "].EmbedTemplate",
				Value:   matcher.EmbedTemplate,
				Details: err.Error(),
			}
		}
		pc.embedMatchersEmbedTemplate[m] = tmpl
		if matcher.ThumbnailURLTemplate != "" {
			if _, err = url.Parse(matcher.ThumbnailURLTemplate); err != nil {
				return &InvalidValueError{
					Field:   "EmbedMatchers[" + m + "].ThumbnailURLTemplate",
					Value:   matcher.ThumbnailURLTemplate,
					Details: err.Error(),
				}
			}
			tmpl, err = template.New(m + "thumb").Parse(matcher.ThumbnailURLTemplate)
			if err != nil {
				return &InvalidValueError{
					Field:   "EmbedMatchers[" + m + "].ThumbnailURLTemplate",
					Value:   matcher.ThumbnailURLTemplate,
					Details: err.Error(),
				}
			}
			pc.embedMatchersThumbnailURLTemplate[m] = tmpl
		}
		if matcher.MediaURLTemplate == "" {
			return &InvalidValueError{
				Field:   "EmbedMatchers[" + m + "].MediaURLTemplate",
				Value:   "",
				Details: "must be set",
			}
		}
		if pc.embedMatchersMediaURLTemplate[m], err = template.New(m + "url").Parse(matcher.MediaURLTemplate); err != nil {
			return &InvalidValueError{
				Field:   "EmbedMatchers[" + m + "].MediaURLTemplate",
				Value:   matcher.MediaURLTemplate,
				Details: err.Error(),
			}
		}
	}
	return nil
}

type UploadConfig struct {
	// MaxFileSize is the maximum allowed file size in bytes for uploads.
	// Default: 15000000 (15 MB)
	MaxFileSize int

	// RejectDuplicateUploads determines whether to reject images and videos that have already been uploaded
	RejectDuplicateUploads bool

	// ThumbWidth is the maximum width that thumbnails in the top thread post will be scaled down to
	// Default: 200
	ThumbWidth int

	// ThumbHeight is the maximum height that thumbnails in the top thread post will be scaled down to
	// Default: 200
	ThumbHeight int

	// ThumbWidthReply is the maximum width that thumbnails in thread replies will be scaled down to
	// Default: 125
	ThumbWidthReply int

	// ThumbHeightReply is the maximum height that thumbnails in thread replies will be scaled down to
	// Default: 125
	ThumbHeightReply int

	// ThumbWidthCatalog is the maximum width that thumbnails on the board catalog page will be scaled down to
	// Default: 50
	ThumbWidthCatalog int

	// ThumbHeightCatalog is the maximum height that thumbnails on the board catalog page will be scaled down to
	// Default: 50
	ThumbHeightCatalog int

	// AllowOtherExtensions is a map of file extensions to use for uploads that are not images or videos
	// The key is the extension (e.g. ".pdf") and the value is the filename of the thumbnail to use in /static
	AllowOtherExtensions map[string]string

	// StripImageMetadata sets what (if any) metadata to remove from uploaded images using exiftool.
	// Valid values are "", "none" (has the same effect as ""), "exif", or "all" (for stripping all metadata)
	StripImageMetadata string
}

func (uc *UploadConfig) AcceptedExtension(filename string) bool {
	ext := strings.ToLower(path.Ext(filename))
	switch ext {
	// images
	case ".gif":
		fallthrough
	case ".jfif":
		fallthrough
	case ".jpeg":
		fallthrough
	case ".jpg":
		fallthrough
	case ".png":
		fallthrough
	case ".webp":
		fallthrough
	// videos
	case ".mp4":
		fallthrough
	case ".webm":
		return true
	}
	// other formats as configured
	_, ok := uc.AllowOtherExtensions[ext]
	return ok
}

// GetBoardConfig returns the custom configuration for the specified board (if it exists)
// or the global board configuration if board is an empty string or it doesn't exist
func GetBoardConfig(board string) *BoardConfig {
	if board == "" {
		return &cfg.BoardConfig
	}
	bc, exists := boardConfigs[board]
	if !exists {
		bc := cfg.BoardConfig
		bc.isGlobal = true
		return &bc
	}
	return &bc
}

func getBoardConfigPath(board string) string {
	// expected to be called with a board when loading the board configuration file for the first time, may or may not exist
	// to be created in the same directory as gochan.json when creating or modifying a board
	if cfg == nil {
		return ""
	}
	board = strings.Trim(board, "/")
	if board == "" {
		return ""
	}

	var paths []string
	paths = append(paths, StandardConfigSearchPaths...)
	for p, cPath := range paths {
		paths[p] = path.Join(path.Dir(cPath), board+"-config.json")
	}
	paths = append(paths, path.Join(cfg.DocumentRoot, board, "board.json"))
	foundPath := gcutil.FindResource(paths...)
	if foundPath == "" {
		return path.Join(path.Dir(cfg.jsonLocation), board+"-config.json")
	}
	return foundPath
}

// ReloadBoardConfig updates or establishes the configuration for the given board
func ReloadBoardConfig(dir string) error {
	boardCfg := GetBoardConfig(dir)
	var boardCfgPath string
	if boardCfg.isGlobal || boardCfg.boardConfigPath == "" {
		// board config hasn't been loaded yet
		boardCfgPath = getBoardConfigPath(dir)
	} else {
		boardCfgPath = boardCfg.boardConfigPath
	}

	if boardCfgPath == "" {
		// this only happens if this is called with an empty string
		return errors.New("no board specified")
	}
	ba, err := os.ReadFile(boardCfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			// board doesn't have a custom config, use global config
			return nil
		}
		return err
	}
	if err = json.Unmarshal(ba, &boardCfg); err != nil {
		return err
	}
	if err = boardCfg.validateEmbedMatchers(); err != nil {
		return err
	}
	boardCfg.isGlobal = false
	boardCfg.boardConfigPath = boardCfgPath
	boardConfigs[dir] = *boardCfg
	return nil
}

// DeleteBoardConfig removes the custom board configuration data, normally should be used
// when a board is deleted
func DeleteBoardConfig(dir string) {
	delete(boardConfigs, dir)
}

// WriteBoardConfig writes the current board configuration to the board's file
func WriteBoardConfig(board string) error {
	if board == "" {
		return errors.New("no board specified")
	}
	bc := GetBoardConfig(board)
	cfgPath := getBoardConfigPath(board)

	fd, err := os.OpenFile(cfgPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, NormalFileMode)
	if err != nil {
		return err
	}
	defer fd.Close()

	if err = json.NewEncoder(fd).Encode(bc); err != nil {
		return err
	}
	return nil
}
