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
	//Rename all existing tables to [name]_old
	var tables = []string{"announcements", "appeals", "banlist", "boards", "embeds", "info", "links", "posts", "reports", "sections", "sessions", "staff", "wordfilters"}
	for _, i := range tables {
		err := renameTable(i, i+"_old")
		if err != nil {
			return err
		}
	}
	var buildfile = "initdb_" + dbType + ".sql"
	//Create all tables for version 1
	err = gcsql.RunSQLFile(gcutil.FindResource("sql/preapril2020migration/" + buildfile))
	if err != nil {
		return err
	}
	//Run data migration
	var migrFile = "oldDBMigration_" + dbType + ".sql"
	err = gcsql.RunSQLFile(gcutil.FindResource("sql/preapril2020migration/"+migrFile,
		"/usr/local/share/gochan/"+migrFile,
		"/usr/share/gochan/"+migrFile))
	if err != nil {
		return err
	}

	//drop all _old tables
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
	err = dropTable("wordfilters_old_normalized")
	if err != nil {
		return err
	}
	return dropTable("numbersequel_temp")
}
