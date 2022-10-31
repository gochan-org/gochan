package pre2021

import (
	"log"

	"github.com/gochan-org/gochan/pkg/gcsql"
)

func (m *Pre2021Migrator) MigrateBoards() error {
	if m.oldBoards == nil {
		m.oldBoards = map[int]string{}
	}
	if m.newBoards == nil {
		m.newBoards = map[int]string{}
	}
	// get all boards from new db
	err := gcsql.ResetBoardSectionArrays()
	if err != nil {
		return nil
	}

	// get boards from old db
	rows, err := m.db.QuerySQL(boardsQuery)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var dir string
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
		if err = rows.Scan(
			&id,
			&dir,
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
		for b := range gcsql.AllBoards {
			if _, ok := m.oldBoards[id]; !ok {
				m.oldBoards[id] = dir
			}
			if gcsql.AllBoards[b].Dir == dir {
				log.Printf("Board /%s/ already exists in new db, moving on\n", dir)
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
			Dir:         dir,
			Title:       title,
			Subtitle:    subtitle,
			Description: description,
			SectionID:   section,
			MaxFilesize: max_file_size,
			// MaxPages:         max_pages,
			DefaultStyle:   default_style,
			Locked:         locked,
			AnonymousName:  anonymous,
			ForceAnonymous: forced_anon,
			// MaxAge:           max_age,
			AutosageAfter:    autosage_after,
			NoImagesAfter:    no_images_after,
			MaxMessageLength: max_message_length,
			AllowEmbeds:      embeds_allowed,
			RedirectToThread: redirect_to_thread,
			RequireFile:      require_file,
			EnableCatalog:    enable_catalog,
		}, false); err != nil {
			return err
		}
		m.newBoards[id] = dir
		log.Printf("/%s/ successfully migrated in the database", dir)
		// Automatic directory migration has the potential to go horribly wrong, so I'm leaving this
		// commented out for now
		// switch m.options.DirAction {
		// case common.DirCopy:

		// case common.DirMove:
		// 	// move the old directory (probably should copy instead) to the new one
		// 	newDocumentRoot := config.GetSystemCriticalConfig().DocumentRoot
		// 	log.Println("Old board path:", path.Join(m.config.DocumentRoot, dir))
		// 	log.Println("Old board path:", path.Join(newDocumentRoot, dir))
		// 	if err = os.Rename(
		// 		path.Join(m.config.DocumentRoot, dir),
		// 		path.Join(newDocumentRoot, dir),
		// 	); err != nil {
		// 		return err
		// 	}
		// 	log.Printf("/%s/ directory/files successfully moved")
		// }
	}
	return nil
}
