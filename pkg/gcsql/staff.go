package gcsql

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path"
	"time"

	"github.com/gochan-org/gochan/pkg/config"
	"github.com/gochan-org/gochan/pkg/gclog"
	"github.com/gochan-org/gochan/pkg/gcutil"
)

// GetStaffName returns the name associated with a session
func GetStaffName(session string) (string, error) {
	const sql = `SELECT staff.username from DBPREFIXstaff as staff
	JOIN DBPREFIXsessions as sessions
	ON sessions.staff_id = staff.id
	WHERE sessions.data = ?`
	var username string
	err := QueryRowSQL(sql, interfaceSlice(session), interfaceSlice(&username))
	return username, err
}

// GetStaffBySession gets the staff that is logged in in the given session
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetStaffBySession(session string) (*Staff, error) {
	const sql = `SELECT 
		staff.id, 
		staff.username, 
		staff.password_checksum, 
		staff.global_rank,
		staff.added_on,
		staff.last_login 
	FROM DBPREFIXstaff as staff
	JOIN DBPREFIXsessions as sessions
	ON sessions.staff_id = staff.id
	WHERE sessions.data = ?`
	staff := new(Staff)
	err := QueryRowSQL(sql, interfaceSlice(session), interfaceSlice(&staff.ID, &staff.Username, &staff.PasswordChecksum, &staff.Rank, &staff.AddedOn, &staff.LastActive))
	return staff, err
}

// EndStaffSession deletes any session rows associated with the requests session cookie and then
// makes the cookie expire, essentially deleting it
func EndStaffSession(writer http.ResponseWriter, request *http.Request) error {
	session, err := request.Cookie("sessiondata")
	if err != nil {
		// No staff session cookie, presumably not logged in so nothing to do
		return nil
	}
	// make it so that the next time the page is loaded, the browser will delete it
	sessionVal := session.Value
	session.MaxAge = 0
	session.Expires = time.Now().Add(-7 * 24 * time.Hour)
	http.SetCookie(writer, session)

	staffID := 0
	if err = QueryRowSQL(`SELECT staff_id FROM DBPREFIXsessions WHERE data = ?`,
		[]interface{}{session.Value}, []interface{}{&staffID}); err != nil && err != sql.ErrNoRows {
		// something went wrong with the query and it's not caused by no rows being returned
		gclog.Printf(gclog.LStaffLog|gclog.LErrorLog,
			"Failed getting staff ID for deletion with cookie data %q", sessionVal)
		return err
	}

	_, err = ExecSQL(`DELETE FROM DBPREFIXsessions WHERE data = ?`, sessionVal)
	if err != nil && err != sql.ErrNoRows {
		gclog.Println(gclog.LStaffLog|gclog.LErrorLog,
			// something went wrong when trying to delete the rows and it's not caused by no rows being returned
			"Failed deleting session for staff with id", staffID)
		return err
	}
	return nil
}

// GetStaffByName gets the staff with a given name
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetStaffByName(name string) (*Staff, error) {
	const sql = `SELECT 
		staff.id, 
		staff.username, 
		staff.password_checksum, 
		staff.global_rank,
		staff.added_on,
		staff.last_login 
	FROM DBPREFIXstaff as staff
	WHERE staff.username = ?`
	staff := new(Staff)
	err := QueryRowSQL(sql, interfaceSlice(name), interfaceSlice(&staff.ID, &staff.Username, &staff.PasswordChecksum, &staff.Rank, &staff.AddedOn, &staff.LastActive))
	return staff, err
}

func getStaffByID(id int) (*Staff, error) {
	const sql = `SELECT 
		staff.id, 
		staff.username, 
		staff.password_checksum, 
		staff.global_rank,
		staff.added_on,
		staff.last_login 
	FROM DBPREFIXstaff as staff
	WHERE staff.id = ?`
	staff := new(Staff)
	err := QueryRowSQL(sql, interfaceSlice(id), interfaceSlice(&staff.ID, &staff.Username, &staff.PasswordChecksum, &staff.Rank, &staff.AddedOn, &staff.LastActive))
	return staff, err
}

// NewStaff creates a new staff account from a given username, password and rank
func NewStaff(username, password string, rank int) error {
	const sql = `INSERT INTO DBPREFIXstaff (username, password_checksum, global_rank)
	VALUES (?, ?, ?)`
	_, err := ExecSQL(sql, username, gcutil.BcryptSum(password), rank)
	return err
}

// DeleteStaff deletes the staff with a given name.
// Implemented to change the account name to a random string and set it to inactive
func DeleteStaff(username string) error {
	const sql = `UPDATE DBPREFIXstaff SET username = ?, is_active = FALSE WHERE username = ?`
	_, err := ExecSQL(sql, gcutil.RandomString(45), username)
	return err
}

func getStaffID(username string) (int, error) {
	staff, err := GetStaffByName(username)
	if err != nil {
		return -1, err
	}
	return staff.ID, nil
}

// CreateSession inserts a session for a given key and username into the database
func CreateSession(key, username string) error {
	const sql1 = `INSERT INTO DBPREFIXsessions (staff_id,data,expires) VALUES(?,?,?)`
	const sql2 = `UPDATE DBPREFIXstaff SET last_login = CURRENT_TIMESTAMP WHERE id = ?`
	staffID, err := getStaffID(username)
	if err != nil {
		return err
	}
	_, err = ExecSQL(sql1, staffID, key, time.Now().Add(time.Duration(time.Hour*730))) //TODO move amount of time to config file
	if err != nil {
		return err
	}
	_, err = ExecSQL(sql2, staffID)
	return err
}

//GetAllStaffNopass gets all staff accounts without their password
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func GetAllStaffNopass(onlyactive bool) ([]Staff, error) {
	sql := `SELECT id, username, global_rank, added_on, last_login FROM DBPREFIXstaff`
	if onlyactive {
		sql += " where is_active = 1"
	}
	rows, err := QuerySQL(sql)
	if err != nil {
		return nil, err
	}
	var staffs []Staff
	for rows.Next() {
		var staff Staff
		err = rows.Scan(&staff.ID, &staff.Username, &staff.Rank, &staff.AddedOn, &staff.LastActive)
		if err != nil {
			return nil, err
		}
		staffs = append(staffs, staff)
	}
	return staffs, nil
}

func createThread(boardID int, locked, stickied, anchored, cyclical bool) (threadID int, err error) {
	const sql = `INSERT INTO DBPREFIXthreads (board_id, locked, stickied, anchored, cyclical) VALUES (?,?,?,?,?)`
	//Retrieves next free ID, explicitly inserts it, keeps retrying until succesfull insert or until a non-pk error is encountered.
	//This is done because mysql doesnt support RETURNING and both LAST_INSERT_ID() and last_row_id() are not thread-safe
	isPrimaryKeyError := true
	for isPrimaryKeyError {
		threadID, err = getNextFreeID("DBPREFIXthreads")
		if err != nil {
			return 0, err
		}
		_, err = ExecSQL(sql, boardID, locked, stickied, anchored, cyclical)

		isPrimaryKeyError, err = errFilterDuplicatePrimaryKey(err)
		if err != nil {
			return 0, err
		}
	}
	return threadID, nil
}

func bumpThreadOfPost(postID int) error {
	id, err := getThreadID(postID)
	if err != nil {
		return err
	}
	return bumpThread(id)
}

func bumpThread(threadID int) error {
	const sql = "UPDATE DBPREFIXthreads SET last_bump = CURRENT_TIMESTAMP WHERE id = ?"
	_, err := ExecSQL(sql, threadID)
	return err
}

func appendFile(postID int, originalFilename, filename, checksum string, fileSize int, isSpoilered bool, width, height, thumbnailWidth, thumbnailHeight int) error {
	const nextIDSQL = `SELECT COALESCE(MAX(file_order) + 1, 0) FROM DBPREFIXfiles WHERE post_id = ?`
	var nextID int
	err := QueryRowSQL(nextIDSQL, interfaceSlice(postID), interfaceSlice(&nextID))
	if err != nil {
		return err
	}
	const insertSQL = `INSERT INTO DBPREFIXfiles (file_order, post_id, original_filename, filename, checksum, file_size, is_spoilered, width, height, thumbnail_width, thumbnail_height)
	VALUES (?,?,?,?,?,?,?,?,?,?,?)`
	_, err = ExecSQL(insertSQL, nextID, postID, originalFilename, filename, checksum, fileSize, isSpoilered, width, height, thumbnailWidth, thumbnailHeight)
	return err
}

//GetThreadIDZeroIfTopPost gets the post id of the top post of the thread a post belongs to, zero if the post itself is the top post
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design. Posts do not directly reference their post post anymore.
func GetThreadIDZeroIfTopPost(postID int) (ID int, err error) {
	const sql = `SELECT t1.id FROM DBPREFIXposts as t1
	JOIN (SELECT thread_id FROM DBPREFIXposts where id = ?) as t2 ON t1.thread_id = t2.thread_id
	WHERE t1.is_top_post`
	err = QueryRowSQL(sql, interfaceSlice(postID), interfaceSlice(&ID))
	if err != nil {
		return 0, err
	}
	if ID == postID {
		return 0, nil
	}
	return ID, nil
}

func getThreadID(postID int) (ID int, err error) {
	const sql = `SELECT thread_id FROM DBPREFIXposts WHERE id = ?`
	err = QueryRowSQL(sql, interfaceSlice(postID), interfaceSlice(&ID))
	return ID, err
}

//GetPostPassword gets the password associated with a given post
func GetPostPassword(postID int) (password string, err error) {
	const sql = `SELECT password FROM DBPREFIXposts WHERE id = ?`
	err = QueryRowSQL(sql, interfaceSlice(postID), interfaceSlice(&password))
	return password, err
}

//UpdatePost updates a post with new information
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design
func UpdatePost(postID int, email, subject string, message template.HTML, messageRaw string) error {
	const sql = `UPDATE DBPREFIXposts SET email = ?, subject = ?, message = ?, message_raw = ? WHERE id = ?`
	_, err := ExecSQL(sql, email, subject, string(message), messageRaw, postID)
	return err
}

//DeleteFilesFromPost deletes all files belonging to a given post
// Deprecated: This method was created to support old functionality during the database refactor of april 2020
// The code should be changed to reflect the new database design. Should be implemented to delete files individually
func DeleteFilesFromPost(postID int) error {
	board, boardWasFound, err := GetBoardFromPostID(postID)
	if err != nil {
		return err
	}
	if !boardWasFound {
		return fmt.Errorf("could not find board for post %v", postID)
	}

	//Get all filenames
	const filenameSQL = `SELECT filename FROM DBPREFIXfiles WHERE post_id = ?`
	rows, err := QuerySQL(filenameSQL, postID)
	if err != nil {
		return err
	}
	var filenames []string
	for rows.Next() {
		var filename string
		if err = rows.Scan(&filename); err != nil {
			return err
		}
		filenames = append(filenames, filename)
	}

	systemCriticalCfg := config.GetSystemCriticalConfig()

	//Remove files from disk
	for _, fileName := range filenames {
		_, filenameBase, fileExt := gcutil.GetFileParts(fileName)

		thumbExt := fileExt
		if thumbExt == "gif" || thumbExt == "webm" || thumbExt == "mp4" {
			thumbExt = "jpg"
		}

		uploadPath := path.Join(systemCriticalCfg.DocumentRoot, board, "/src/", filenameBase+"."+fileExt)
		thumbPath := path.Join(systemCriticalCfg.DocumentRoot, board, "/thumb/", filenameBase+"t."+thumbExt)
		catalogThumbPath := path.Join(systemCriticalCfg.DocumentRoot, board, "/thumb/", filenameBase+"c."+thumbExt)

		os.Remove(uploadPath)
		os.Remove(thumbPath)
		os.Remove(catalogThumbPath)
	}

	const removeFilesSQL = `DELETE FROM DBPREFIXfiles WHERE post_id = ?`
	_, err = ExecSQL(removeFilesSQL, postID)
	return err
}

//DeletePost deletes a post with a given ID
func DeletePost(postID int, checkIfTopPost bool) error {
	if checkIfTopPost {
		isTopPost, err := isTopPost(postID)
		if err != nil {
			return err
		}
		if isTopPost {
			threadID, err := getThreadID(postID)
			if err != nil {
				return err
			}
			return deleteThread(threadID)
		}
	}

	DeleteFilesFromPost(postID)
	const sql = `UPDATE DBPREFIXposts SET is_deleted = TRUE, deleted_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := ExecSQL(sql, postID)
	return err
}

func isTopPost(postID int) (val bool, err error) {
	const sql = `SELECT is_top_post FROM DBPREFIXposts WHERE id = ?`
	err = QueryRowSQL(sql, interfaceSlice(postID), interfaceSlice(&val))
	return val, err
}

func deleteThread(threadID int) error {
	const sql1 = `UPDATE DBPREFIXthreads SET is_deleted = TRUE, deleted_at = CURRENT_TIMESTAMP WHERE id = ?`
	const sql2 = `SELECT id FROM DBPREFIXposts WHERE thread_id = ?`

	_, err := QuerySQL(sql1, threadID)
	if err != nil {
		return err
	}
	rows, err := QuerySQL(sql2, threadID)
	if err != nil {
		return err
	}
	var ids []int
	for rows.Next() {
		var id int
		if err = rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}

	for _, id := range ids {
		if err = DeletePost(id, false); err != nil {
			return err
		}
	}
	return nil
}

func createUser(username, passwordEncrypted string, globalRank int) (userID int, err error) {
	const sqlInsert = `INSERT INTO DBPREFIXstaff (username, password_checksum, global_rank) VALUES (?,?,?)`
	const sqlSelect = `SELECT id FROM DBPREFIXstaff WHERE username = ?`
	//Excecuted in two steps this way because last row id functions arent thread safe, username is unique
	_, err = ExecSQL(sqlInsert, username, passwordEncrypted, globalRank)
	if err != nil {
		return 0, err
	}
	err = QueryRowSQL(sqlSelect, interfaceSlice(username), interfaceSlice(&userID))
	return userID, err
}
