package pre2021

import (
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcsql"
)

func (m *Pre2021Migrator) MigrateBoards() error {
	// get all boards from new db
	boards, err := gcsql.GetAllBoards()
	if err != nil {
		return err
	}

	// get boards from old db
	rows, err := m.db.QuerySQL(`SELECT
	id,
	dir,
	type,
	upload_type,
	title,
	subtitle,
	description,
	section,
	max_file_size,
	max_pages,
	default_style,
	locked,
	anonymous,
	forced_anon,
	max_age,
	autosage_after,
	no_images_after,
	max_message_length,
	embeds_allowed,
	redirect_to_thread,
	require_file,
	enable_catalog
	FROM DBPREFIXboards`)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int
		var dir string
		var board_type int
		var upload_type int
		var title string
		var subtitle string
		var description string
		var section int
		var max_file_size int
		var max_pages int
		var default_style string
		var locked bool
		var anonymous string
		var forced_anon bool
		var max_age int
		var autosage_after int
		var no_images_after int
		var max_message_length int
		var embeds_allowed bool
		var redirect_to_thread bool
		var require_file bool
		var enable_catalog bool
		if err = rows.Scan(&dir,
			&board_type,
			&upload_type,
			&title,
			&subtitle,
			&description,
			&section,
			&max_file_size,
			&max_pages,
			&default_style,
			&locked,
			&anonymous,
			&forced_anon,
			&max_age,
			&autosage_after,
			&no_images_after,
			&max_message_length,
			&embeds_allowed,
			&redirect_to_thread,
			&require_file,
			&enable_catalog); err != nil {
			return err
		}
		found := false
		for _, board := range boards {
			if _, ok := m.oldBoards[id]; !ok {
				m.oldBoards[id] = dir
			}
			if board.Dir == dir {
				gclog.Printf(gclog.LStdLog, "Board /%s/ already exists in new db, moving on\n", dir)
				found = true
				break
			}
		}
		if found {
			continue
		}
		// create new board using the board data from the old db
		// omitting things like ID and creation date since we don't really care
		if err = gcsql.CreateBoard(&gcsql.Board{
			Dir:              dir,
			Type:             board_type,  // ??
			UploadType:       upload_type, // ??
			Title:            title,
			Subtitle:         subtitle,
			Description:      description,
			Section:          section,
			MaxFilesize:      max_file_size,
			MaxPages:         max_pages,
			DefaultStyle:     default_style,
			Locked:           locked,
			Anonymous:        anonymous,
			ForcedAnon:       forced_anon,
			MaxAge:           max_age,
			AutosageAfter:    autosage_after,
			NoImagesAfter:    no_images_after,
			MaxMessageLength: max_message_length,
			EmbedsAllowed:    embeds_allowed,
			RedirectToThread: redirect_to_thread,
			RequireFile:      require_file,
			EnableCatalog:    enable_catalog,
		}); err != nil {
			return err
		}
		m.newBoards[id] = dir
		gclog.Printf(gclog.LStdLog, "/%s/ successfully migrated in the database")
		// switch m.options.DirAction {
		// case common.DirCopy:

		// case common.DirMove:
		// 	// move the old directory (probably should copy instead) to the new one
		// 	newDocumentRoot := config.GetSystemCriticalConfig().DocumentRoot
		// 	gclog.Println(gclog.LStdLog, "Old board path:", path.Join(m.config.DocumentRoot, dir))
		// 	gclog.Println(gclog.LStdLog, "Old board path:", path.Join(newDocumentRoot, dir))
		// 	if err = os.Rename(
		// 		path.Join(m.config.DocumentRoot, dir),
		// 		path.Join(newDocumentRoot, dir),
		// 	); err != nil {
		// 		return err
		// 	}
		// 	gclog.Printf(gclog.LStdLog, "/%s/ directory/files successfully moved")
		// }
	}
	return nil
}
