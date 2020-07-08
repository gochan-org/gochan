package gcmigrate

import (
	"regexp"
	"strconv"

	"github.com/gochan-org/gochan/pkg/gcsql"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

func migratePreApril2020Database(dbType string) error {
	_, err := gcsql.ExecSQL("DROP TABLE IF EXISTS DBPREFIXsessions")
	if err != nil {
		return err
	}
	err = createNumberSequelTable(1000) //number sequel table is used in normalizing comma seperated lists
	if err != nil {
		return err
	}
	//Rename all existing tables to [name]_old
	var tables = []string{"announcements", "appeals", "banlist", "boards", "embeds", "info", "links", "posts", "reports", "sections", "staff", "wordfilters"}
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

	//Fix posts linking
	err = fixPostLinking()
	if err != nil {
		return err
	}

	//Remove old self id on posts
	_, err = gcsql.ExecSQL("ALTER TABLE DBPREFIXposts DROP COLUMN oldselfid")
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
	return dropTable("numbersequel_temp")
}

func fixPostLinking() error {
	boards, err := getAllBoardIds()
	if err != nil {
		return err
	}
	for _, boardID := range boards {
		err = fixPostLinkingOnBoard(boardID)
		if err != nil {
			return err
		}
	}
	return nil
}

func fixPostLinkingOnBoard(boardID int) error {
	//Create jumptable of old id -> new id of all posts
	rows, err := gcsql.QuerySQL(`SELECT oldselfid, posts.id FROM DBPREFIXposts as posts
	JOIN DBPREFIXthreads as threads ON threads.id = posts.thread_id 
	WHERE threads.board_id = ?`, boardID)
	if err != nil {
		return err
	}
	jumptable := make(map[int]int)
	for rows.Next() {
		var old int
		var new int
		err = rows.Scan(&old, &new)
		if err != nil {
			return err
		}
		jumptable[old] = new
	}

	jumpTableFunc := func(intstring string) string {
		oldValue, err := strconv.Atoi(intstring)
		if err != nil {
			print(err.Error())
		}
		if newintvalue, ok := jumptable[oldValue]; ok {
			return strconv.Itoa(newintvalue)
		}
		return "(unkown post id during migration)"
	}

	replaceNumbers := func(input string) string {
		pattern := regexp.MustCompile("[0-9]+")
		return replaceAllStringSubmatchFunc(pattern, input, jumpTableFunc)
	}

	//get all unformatted text which contain >>
	messagesWithPossibleNumbers, err := getAllLinkingRawText(boardID)
	if err != nil {
		return err
	}

	// Regex pattern captures the numbers following >>
	// pattern1 matches any occurance with a whitespace (tab, space, linebreak) preceeding >>, with a whitepace or EOF after the numbers
	// pattern2 matches any occurance with a file start preceeding >>, with a whitepace or EOF after the numbers
	pattern1 := regexp.MustCompile(`\s>>[0-9]+(?:\s|$)`)
	pattern2 := regexp.MustCompile(`^>>[0-9]+(?:\s|$)`)
	for i := range messagesWithPossibleNumbers {
		messagesWithPossibleNumbers[i].MessageRaw = replaceAllStringSubmatchFunc(pattern1, messagesWithPossibleNumbers[i].MessageRaw, replaceNumbers)
		messagesWithPossibleNumbers[i].MessageRaw = replaceAllStringSubmatchFunc(pattern2, messagesWithPossibleNumbers[i].MessageRaw, replaceNumbers)
	}

	//Save reformatted text
	err = setUnformattedInDatabase(messagesWithPossibleNumbers)
	if err != nil {
		return err
	}
	return nil
}

//replaceAllStringSubmatchFunc replaces all matches in a substring with the result of putting that match through a given string to string function
func replaceAllStringSubmatchFunc(re *regexp.Regexp, input string, function func(s string) string) string {
	matches := re.FindAllStringSubmatchIndex(input, -1)
	if len(matches) == 0 {
		return input
	}
	output := ""
	oldPairEnd := 0
	for _, pair := range matches {
		output += input[oldPairEnd:pair[0]]
		output += function(input[pair[0]:pair[1]])
		oldPairEnd = pair[1]
	}
	output += input[oldPairEnd:]
	return output
}
