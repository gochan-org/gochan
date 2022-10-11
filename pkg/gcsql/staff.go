package gcsql

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gochan-org/gochan/pkg/gcutil"
)

// createDefaultAdminIfNoStaff creates a new default admin account if no accounts exist
func createDefaultAdminIfNoStaff() error {
	const sql = `SELECT COUNT(id) FROM DBPREFIXstaff`
	var count int
	QueryRowSQL(sql, interfaceSlice(), interfaceSlice(&count))
	if count > 0 {
		return nil
	}
	_, err := NewStaff("admin", "password", 3)
	return err
}

func NewStaff(username string, password string, rank int) (*Staff, error) {
	const sqlINSERT = `INSERT INTO DBPREFIXstaff
	(username, password_checksum, global_rank)
	VALUES(?,?,?)`
	passwordChecksum := gcutil.BcryptSum(password)
	_, err := ExecSQL(sqlINSERT, username, passwordChecksum, rank)
	if err != nil {
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
	const updateActive = `UPDATE DBPREFIXstaff SET is_active = 0 WHERE username = ?`
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
		if err = QueryRowSQL(query, interfaceSlice(s.Username), interfaceSlice(&s.ID)); err != nil {
			return err
		}
	}
	_, err = ExecSQL(deleteSessions, s.ID)
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

// GetStaffBySession gets the staff that is logged in in the given session
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
	err := QueryRowSQL(sql, interfaceSlice(session), interfaceSlice(&staff.ID, &staff.Username, &staff.PasswordChecksum, &staff.Rank, &staff.AddedOn, &staff.LastLogin))
	return staff, err
}
