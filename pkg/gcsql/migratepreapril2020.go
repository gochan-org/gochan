package gcsql

import "github.com/gochan-org/gochan/pkg/gcutil"

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
	err = runSQLFile(gcutil.FindResource(buildfile))
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
	//return dropNumberSequelTable()
	return nil
}

func createNumberSequelTable(count int) error {
	_, err := ExecSQL("CREATE TABLE DBPREFIXnumbersequel_temp(num INT)")
	if err != nil {
		return err
	}
	for i := 1; i < count; i++ {
		_, err = ExecSQL(`INSERT INTO DBPREFIXnumbersequel_temp(num) VALUES (?)`, i)
		if err != nil {
			return err
		}
	}
	return nil
}

func dropNumberSequelTable() error {
	_, err := ExecSQL("DROP TABLE DBPREFIXnumbersequel_temp")
	return err
}
