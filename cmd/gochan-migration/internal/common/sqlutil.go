package common

import (
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

var (
	commentRemover = regexp.MustCompile("--.*\n?")
)

func RunSQLFile(path string, db *gcsql.GCDB) error {
	sqlBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	sqlStr := commentRemover.ReplaceAllString(string(sqlBytes), " ")
	sqlArr := strings.Split(sqlStr, ";")

	for _, statement := range sqlArr {
		statement = strings.Trim(statement, " \n\r\t")
		if len(statement) > 0 {
			if _, err = db.ExecSQL(statement); err != nil {
				return err
			}
		}
	}
	return nil
}

func InitDB(initFile string, db *gcsql.GCDB) error {
	filePath := gcutil.FindResource(initFile,
		"/usr/local/share/gochan/"+initFile,
		"/usr/share/gochan/"+initFile)
	if filePath == "" {
		return fmt.Errorf(
			"SQL database initialization file (%s) missing. Please reinstall gochan-migration", initFile)
	}

	return RunSQLFile(filePath, db)
}
