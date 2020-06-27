package gcmigrate

import (
	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

func migratePreApril2020Database(dbType string) error {
	err := createNumberSequelTable(1000) //number sequel table is used in normalizing comma seperated lists
	if err != nil {
		return err
	}
	var tables = []string{"announcements", "appeals", "banlist", "boards", "embeds", "info", "links", "posts", "reports", "sections", "sessions", "staff", "wordfilters"}
	for _, i := range tables {
		err := renameTable(i, i+"_old")
		if err != nil {
			return err
		}
	}
	var buildfile = "initdb_" + dbType + ".sql"
	//err := runSQLFile(gcutil.FindResource("sql/preapril2020migration/" + buildfile)) //TODO move final version 1 build script next to migrate script and exec that
	err = gcsql.RunSQLFile(gcutil.FindResource(buildfile))
	if err != nil {
		return err
	}
	var migrFile = "oldDBMigration_" + dbType + ".sql"
	err = gcsql.RunSQLFile(gcutil.FindResource("sql/preapril2020migration/"+migrFile,
		"/usr/local/share/gochan/"+migrFile,
		"/usr/share/gochan/"+migrFile))
	if err != nil {
		return err
	}

	for _, i := range tables {
		err := dropTable(i + "_old")
		if err != nil {
			return err
		}
	}
	err = dropTable("banlist_old_normalized")
	if err != nil {
		return err
	}
	return dropNumberSequelTable()
}
