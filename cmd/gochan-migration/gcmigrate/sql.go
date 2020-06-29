package gcmigrate

import "github.com/gochan-org/gochan/pkg/gcsql"

func renameTable(tablename string, tableNameNew string) error {
	var sql = "ALTER TABLE DBPREFIX" + tablename + " RENAME TO DBPREFIX" + tableNameNew
	_, err := gcsql.ExecSQL(sql)
	return err
}

func dropTable(tablename string) error {
	var sql = "DROP TABLE DBPREFIX" + tablename
	_, err := gcsql.ExecSQL(sql)
	return err
}

func createNumberSequelTable(count int) error {
	_, err := gcsql.ExecSQL("CREATE TABLE DBPREFIXnumbersequel_temp(num INT)")
	if err != nil {
		return err
	}
	for i := 1; i < count; i++ {
		_, err = gcsql.ExecSQL(`INSERT INTO DBPREFIXnumbersequel_temp(num) VALUES (?)`, i)
		if err != nil {
			return err
		}
	}
	return nil
}
