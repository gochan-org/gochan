package gcsql

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gochan-org/gochan/pkg/gcutil"
)

var (
	ErrUnrecognizedUsername = errors.New("invalid username")
)

// createDefaultAdminIfNoStaff creates a new default admin account if no accounts exist
func createDefaultAdminIfNoStaff() error {
	const query = `SELECT COUNT(id) FROM DBPREFIXstaff`
	var count int
	err := QueryRowSQL(query, nil, []any{&count})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	if count > 0 {
		return nil
	}
	_, err = NewStaff("admin", "password", 3)
	return err
}

func NewStaff(username string, password string, rank int) (*Staff, error) {
	const sqlINSERT = `INSERT INTO DBPREFIXstaff
	(username, password_checksum, global_rank)
	VALUES(?,?,?)`
	passwordChecksum := gcutil.BcryptSum(password)
	_, err := ExecSQL(sqlINSERT, username, passwordChecksum, rank)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}
	return &Staff{
		Username:         username,
		PasswordChecksum: passwordChecksum,
		Rank:             rank,
		AddedOn:          time.Now(),
		IsActive:         true,
	}, nil
}

// SetActive changes the active status of the staff member. If `active` is false, the login sessions are cleared
func (s *Staff) SetActive(active bool) error {
	const updateActive = `UPDATE DBPREFIXstaff SET is_active = FALSE WHERE username = ?`
	_, err := ExecSQL(updateActive, s.Username)
	if err != nil {
		return err
	}
	if active {
		return nil
	}
	return s.ClearSessions()
}

// ClearSessions clears all login sessions for the user, requiring them to login again
func (s *Staff) ClearSessions() error {
	const query = `SELECT id FROM DBPREFIXstaff WHERE username = ?`
	const deleteSessions = `DELETE FROM DBPREFIXsessions WHERE staff_id = ?`
	var err error
	if s.ID == 0 {
		// ID field not set, get it from the DB
		if err = QueryRowSQL(query, []any{s.Username}, []any{&s.ID}); err != nil {
			return err
		}
	}
	_, err = ExecSQL(deleteSessions, s.ID)
	return err
}

func (s *Staff) RankTitle() string {
	if s.Rank == 3 {
		return "Administrator"
	} else if s.Rank == 2 {
		return "Moderator"
	} else if s.Rank == 1 {
		return "Janitor"
	}
	return ""
}

func UpdatePassword(username string, newPassword string) error {
	const sqlUPDATE = `UPDATE DBPREFIXstaff SET password_checksum = ? WHERE username = ?`
	checksum := gcutil.BcryptSum(newPassword)
	_, err := ExecSQL(sqlUPDATE, checksum, username)
	return err
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
	session.MaxAge = -1
	http.SetCookie(writer, session)

	staffID := 0
	if err = QueryRowSQL(`SELECT staff_id FROM DBPREFIXsessions WHERE data = ?`,
		[]interface{}{session.Value}, []interface{}{&staffID}); err != nil && err != sql.ErrNoRows {
		// something went wrong with the query and it's not caused by no rows being returned
		return errors.New("failed getting staff ID: " + err.Error())
	}

	_, err = ExecSQL(`DELETE FROM DBPREFIXsessions WHERE data = ?`, sessionVal)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed clearing session for staff with id %d", staffID)
	}
	return nil
}

func DeactivateStaff(username string) error {
	s := Staff{Username: username}
	return s.SetActive(false)
}

func GetStaffUsernameFromID(id int) (string, error) {
	const query = `SELECT username FROM DBPREFIXstaff WHERE id = ?`
	var username string
	err := QueryRowSQL(query, []any{id}, []any{&username})
	return username, err
}

func GetStaffID(username string) (int, error) {
	const query = `SELECT id  FROM DBPREFIXstaff WHERE username = ?`
	var id int
	err := QueryRowSQL(query, []any{username}, []any{&id})
	return id, err
}

// GetStaffBySession gets the staff that is logged in in the given session
func GetStaffBySession(session string) (*Staff, error) {
	const query = `SELECT 
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
	err := QueryRowSQL(query, []any{session}, []any{
		&staff.ID, &staff.Username, &staff.PasswordChecksum, &staff.Rank, &staff.AddedOn, &staff.LastLogin})
	return staff, err
}

func GetStaffByUsername(username string, onlyActive bool) (*Staff, error) {
	query := `SELECT 
	id, username, password_checksum, global_rank, added_on, last_login, is_active
	FROM DBPREFIXstaff WHERE username = ?`
	if onlyActive {
		query += ` AND is_active = TRUE`
	}
	staff := new(Staff)
	err := QueryRowSQL(query, []any{username}, []any{
		&staff.ID, &staff.Username, &staff.PasswordChecksum, &staff.Rank, &staff.AddedOn,
		&staff.LastLogin, &staff.IsActive,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUnrecognizedUsername
	}
	return staff, err
}

// CreateLoginSession inserts a session for a given key and username into the database
func (staff *Staff) CreateLoginSession(key string) error {
	const insertSQL = `INSERT INTO DBPREFIXsessions (staff_id,data,expires) VALUES(?,?,?)`
	const updateSQL = `UPDATE DBPREFIXstaff SET last_login = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := ExecSQL(insertSQL, staff.ID, key, time.Now().Add(time.Duration(time.Hour*730))) //TODO move amount of time to config file
	if err != nil {
		return err
	}
	_, err = ExecSQL(updateSQL, staff.ID)
	return err
}
