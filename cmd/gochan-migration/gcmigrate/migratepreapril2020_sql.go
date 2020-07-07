package gcmigrate

import (
	"github.com/gochan-org/gochan/pkg/gcsql"
)

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

type rawMessageWithID struct {
	ID         int
	MessageRaw string
}

func getAllLinkingRawText(boardID int) ([]rawMessageWithID, error) {
	const sql = `SELECT posts.id, posts.message_raw FROM DBPREFIXposts AS posts
	JOIN DBPREFIXthreads AS threads ON posts.thread_id = threads.id
	WHERE posts.message_raw LIKE '%>>%' AND threads.board_id = ?`
	rows, err := gcsql.QuerySQL(sql, boardID)
	if err != nil {
		return nil, err
	}
	var messages []rawMessageWithID
	for rows.Next() {
		var message rawMessageWithID
		if err = rows.Scan(&message.ID, &message.MessageRaw); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	return messages, nil
}

func setUnformattedInDatabase(messages []rawMessageWithID) error {
	const sql = `UPDATE DBPREFIXposts
	SET message_raw = ?
	WHERE id = ?`
	stmt, err := gcsql.PrepareSQL(sql)
	defer stmt.Close()
	if err != nil {
		return err
	}
	for _, message := range messages {
		if _, err = stmt.Exec(string(message.MessageRaw), message.ID); err != nil {
			return err
		}
	}
	return err
}

func getAllBoardIds() ([]int, error) {
	const sql = `SELECT id FROM DBPREFIXboards`
	rows, err := gcsql.QuerySQL(sql)
	if err != nil {
		return nil, err
	}
	var boards []int
	for rows.Next() {
		var board int
		err = rows.Scan(&board)
		if err != nil {
			return nil, err
		}
		boards = append(boards, board)
	}
	return boards, nil
}
