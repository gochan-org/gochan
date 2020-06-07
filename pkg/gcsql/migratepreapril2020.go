package gcsql

import "github.com/gochan-org/gochan/pkg/gcutil"

func migratePreApril2020Database(dbType string) error {
	var tables = []string{"announcements", "appeals", "banlist", "boards", "embeds", "info", "links", "posts", "reports", "sections", "sessions", "staff", "wordfilters"}
	for _, i := range tables {
		err := renameTable(i, i+"_old")
		if err != nil {
			return err
		}
	}
	var buildfile = "initdb_" + dbType + ".sql"
	err := runSQLFile(gcutil.FindResource("sql/preapril2020migration/" + buildfile))
	if err != nil {
		return err
	}
	var migrFile = "oldDBMigration_" + dbType + ".sql"
	err = runSQLFile(gcutil.FindResource("sql/preapril2020migration/"+migrFile,
		"/usr/local/share/gochan/"+migrFile,
		"/usr/share/gochan/"+migrFile))
	if err != nil {
		return err
	}

	// for _, i := range tables {
	// 	err := dropTable(i + "_old")
	// 	if err != nil {
	// 		return err
	// 	}
	// }
	return nil
}
