package gcupdate

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"time"

	"github.com/gochan-org/gochan/cmd/gochan-migration/internal/common"
	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/rs/zerolog"
)

// WordFilter represents data in the deprecated wordfilters table
type Wordfilter struct {
	ID        int       // sql: `id`
	BoardDirs *string   // sql: `board_dirs`
	BoardID   *int      // sql: `board_id`, replaced with board_dirs
	StaffID   int       // sql: `staff_id`
	StaffNote string    // sql: `staff_note`
	IssuedAt  time.Time // sql: `issued_at`
	Search    string    // sql: `search`
	IsRegex   bool      // sql: `is_regex`
	ChangeTo  string    // sql: `change_to`
}

type filenameOrUsernameBanBase struct {
	ID        int       // sql: id
	BoardID   *int      // sql: board_id
	StaffID   int       // sql: staff_id
	StaffNote string    // sql: staff_note
	IssuedAt  time.Time // sql: issued_at
	check     string    // replaced with username or filename
	IsRegex   bool      // sql: is_regex
}

// UsernameBan represents data in the deprecated username_ban table
type UsernameBan struct {
	filenameOrUsernameBanBase
	Username string // sql: `username`
}

// FilenameBan represents data in the deprecated filename_ban table
type FilenameBan struct {
	filenameOrUsernameBanBase
	Filename string // sql: `filename`
	IsRegex  bool   // sql: `is_regex`
}

// FileBan represents data in the deprecated file_ban table
type FileBan struct {
	ID            int       // sql: `id`
	BoardID       *int      // sql: `board_id`
	StaffID       int       // sql: `staff_id`
	StaffNote     string    // sql: `staff_note`
	IssuedAt      time.Time // sql: `issued_at`
	Checksum      string    // sql: `checksum`
	Fingerprinter *string   // sql: `fingerprinter`
	BanIP         bool      // sql: `ban_ip`
	BanIPMessage  *string   // sql: `ban_ip_message`
}

// addFilterTables is used for the db version 4 upgrade to create the filter tables from the respective SQL init file
func addFilterTables(ctx context.Context, db *gcsql.GCDB, tx *sql.Tx, sqlConfig *config.SQLConfig, errEv *zerolog.Event) error {
	filePath, err := common.GetInitFilePath("initdb_" + sqlConfig.DBtype + ".sql")
	defer func() {
		if err != nil {
			errEv.Err(err).Caller(1).Send()
		}
	}()
	if err != nil {
		return err
	}
	ba, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	sqlStr := common.CommentRemover.ReplaceAllString(string(ba), " ")
	sqlArr := strings.Split(sqlStr, ";")

	for _, stmtStr := range sqlArr {
		stmtStr = strings.TrimSpace(stmtStr)
		if !strings.HasPrefix(stmtStr, "CREATE TABLE DBPREFIXfilter") {
			continue
		}
		if _, err = db.ExecContextSQL(ctx, tx, stmtStr); err != nil {
			return err
		}
	}
	return nil
}
