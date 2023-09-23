package database

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/glebarez/go-sqlite"
)

func CreateEvent(event Event) error {
	db, err := sql.Open("sqlite", "/home/pi/climbing-society-seats-app/database.db")
	if err != nil {
		return err
	}
	defer db.Close()

	query := "INSERT INTO events (event_location, event_date, meet_location, meet_time, total_seats, require_member, open_datetime, close_datetime) VALUES (?, ?, ?, ?, ?, ?, ?, ?)"
	_, err = db.Exec(query, event.EventLocation, event.EventDate, event.MeetLocation, event.MeetTime, event.TotalSeats, event.RequireMember, event.OpenDatetime, event.CloseDatetime)
	if err != nil {
		return err
	}

	return nil
}

func DeleteEvent(eventId int) error {
	db, err := sql.Open("sqlite", "/home/pi/climbing-society-seats-app/database.db")
	if err != nil {
		return err
	}
	defer db.Close()

	query := "DELETE FROM events WHERE event_id = ?"
	_, err = db.Exec(query, eventId)
	if err != nil {
		return err
	}

	// Cascade delete
	query = "DELETE FROM participants WHERE event_id = ?"
	_, err = db.Exec(query, eventId)
	if err != nil {
		return err
	}

	return nil
}

type Event struct {
	EventID       int         `db:"event_id" json:"event_id"`
	EventLocation string      `db:"event_location" json:"session_location"`
	EventDate     string      `db:"event_date" json:"session_date"`
	MeetLocation  string      `db:"meet_location" json:"meet_point"`
	MeetTime      string      `db:"meet_time" json:"meet_time"`
	TotalSeats    int         `db:"total_seats" json:"total_seats"`
	SeatsTaken    int         `db:"seats_taken" json:"current_seats"`
	RequireMember bool        `db:"require_member" json:"require_member"`
	OpenDatetime  string      `db:"open_datetime" json:"open_date"`
	CloseDatetime string      `db:"close_datetime" json:"close_date"`
	EventStatus   EventStatus `db:"event_status"`
}

type EventStatus int

const (
	EventStatusScheduled EventStatus = iota
	EventStatusClosed
)

func (e *Event) Close() error {
	e.EventStatus = EventStatusClosed
	return UpdateEventInDatabase(e.EventID, *e)
}

func (e *Event) GetLink() string {
	return fmt.Sprintf("http://uowclimbingsociety.tplinkdns.com:8080/register?event=%d", e.EventID)
}

func (e *Event) GetParticipants() ([]Participant, error) {
	return GetEventParticipants(e.EventID)
}

func (e *Event) AddParticipant(firstName string, surname string, member bool) error {
	// Confirm there is space
	if e.TotalSeats-e.SeatsTaken <= 0 {
		return errors.New("no seats available")
	}

	// Create DB connection
	db, err := sql.Open("sqlite", "/home/pi/climbing-society-seats-app/database.db")
	if err != nil {
		return err
	}
	defer db.Close()

	// Check if the participant name already exists for the event
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM participants WHERE event_id = ? AND first_name = ? AND surname = ?", e.EventID, firstName, surname).Scan(&count)
	if err != nil {
		return fmt.Errorf("failed to execute SELECT statement: %v", err)
	}

	if count > 0 {
		return fmt.Errorf("participant name already exists for the event")
	}

	// Add participant
	stmt, err := db.Prepare("INSERT INTO participants (event_id, first_name, surname, member) VALUES (?, ?, ?, ?)")
	if err != nil {
		return fmt.Errorf("failed to prepare INSERT statement: %v", err)
	}
	defer stmt.Close()

	_, err = stmt.Exec(e.EventID, firstName, surname, member)
	if err != nil {
		return fmt.Errorf("failed to execute INSERT statement: %v", err)
	}

	// Update seats taken
	updateStmt, err := db.Prepare("UPDATE events SET seats_taken = seats_taken + 1 WHERE event_id = ?")
	if err != nil {
		return fmt.Errorf("failed to prepare UPDATE statement: %v", err)
	}
	defer updateStmt.Close()

	_, err = updateStmt.Exec(e.EventID)
	if err != nil {
		return fmt.Errorf("failed to execute UPDATE statement: %v", err)
	}

	return nil
}

func GetEventByID(eventID int) (*Event, error) {
	db, err := sql.Open("sqlite", "/home/pi/climbing-society-seats-app/database.db")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := "SELECT event_id, event_location, event_date, meet_location, meet_time, total_seats, seats_taken, require_member, open_datetime, close_datetime, event_status FROM events WHERE event_id = ?"
	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(eventID)

	var event Event
	err = row.Scan(
		&event.EventID,
		&event.EventLocation,
		&event.EventDate,
		&event.MeetLocation,
		&event.MeetTime,
		&event.TotalSeats,
		&event.SeatsTaken,
		&event.RequireMember,
		&event.OpenDatetime,
		&event.CloseDatetime,
		&event.EventStatus,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("event not found")
		}
		return nil, err
	}

	return &event, nil
}

type Participant struct {
	ParticipantID int    `db:"participant_id" json:"participant_id"`
	EventID       int    `db:"event_id" json:"event_id"`
	FirstName     string `db:"first_name" json:"first_name"`
	LastName      string `db:"surname" json:"last_name"`
	Member        bool   `db:"member" json:"member"`
}

func GetEventParticipants(eventID int) ([]Participant, error) {
	db, err := sql.Open("sqlite", "/home/pi/climbing-society-seats-app/database.db")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := "SELECT participant_id, event_id, first_name, surname, member FROM participants WHERE event_id = ?"
	rows, err := db.Query(query, eventID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	participants := []Participant{}
	for rows.Next() {
		var participant Participant
		if err := rows.Scan(
			&participant.ParticipantID,
			&participant.EventID,
			&participant.FirstName,
			&participant.LastName,
			&participant.Member,
		); err != nil {
			return nil, err
		}
		participants = append(participants, participant)
	}

	return participants, nil
}

func GetParticipantByID(participantID int) (*Participant, error) {
	db, err := sql.Open("sqlite", "/home/pi/climbing-society-seats-app/database.db")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := "SELECT participant_id, event_id, first_name, surname, member FROM participants WHERE participant_id = ?"
	stmt, err := db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	row := stmt.QueryRow(participantID)

	var participant Participant
	err = row.Scan(
		&participant.ParticipantID,
		&participant.EventID,
		&participant.FirstName,
		&participant.LastName,
		&participant.Member,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("participant not found")
		}
		return nil, err
	}

	return &participant, nil
}

func DeleteParticipant(participantID int) error {
	db, err := sql.Open("sqlite", "/home/pi/climbing-society-seats-app/database.db")
	if err != nil {
		return err
	}
	defer db.Close()

	participant, err := GetParticipantByID(participantID)
	if err != nil {
		return err
	}

	event, err := GetEventByID(participant.EventID)
	if err != nil {
		return err
	}

	query := "DELETE FROM participants WHERE participant_id = ?"
	res, err := db.Exec(query, participantID)
	fmt.Println(res)
	if err != nil {
		return err
	}

	event.SeatsTaken = event.SeatsTaken - 1
	err = UpdateEventInDatabase(event.EventID, *event)
	if err != nil {
		return err
	}

	return nil
}

func UpdateEventInDatabase(eventID int, eventData Event) error {
	db, err := sql.Open("sqlite", "/home/pi/climbing-society-seats-app/database.db")
	if err != nil {
		return err
	}
	defer db.Close()

	query := `
        UPDATE events
        SET
            event_location = ?,
            event_date = ?,
            meet_location = ?,
            meet_time = ?,
            total_seats = ?,
			seats_taken = ?,
            require_member = ?,
            open_datetime = ?,
            close_datetime = ?,
			event_status = ?
        WHERE event_id = ?
    `

	_, err = db.Exec(
		query,
		eventData.EventLocation,
		eventData.EventDate,
		eventData.MeetLocation,
		eventData.MeetTime,
		eventData.TotalSeats,
		eventData.SeatsTaken,
		eventData.RequireMember,
		eventData.OpenDatetime,
		eventData.CloseDatetime,
		eventData.EventStatus,
		eventID,
	)
	if err != nil {
		return err
	}

	return nil
}

func GetEvents() ([]Event, error) {
	db, err := sql.Open("sqlite", "/home/pi/climbing-society-seats-app/database.db")
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to database: %s", err)
	}
	defer db.Close()

	query := "SELECT event_id, event_location, event_date, meet_location, meet_time, total_seats, seats_taken, require_member, open_datetime, close_datetime, event_status FROM events"
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("Failed to get events: %s", err)
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var event Event
		err = rows.Scan(
			&event.EventID,
			&event.EventLocation,
			&event.EventDate,
			&event.MeetLocation,
			&event.MeetTime,
			&event.TotalSeats,
			&event.SeatsTaken,
			&event.RequireMember,
			&event.OpenDatetime,
			&event.CloseDatetime,
			&event.EventStatus,
		)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse event: %s", err)
		}
		events = append(events, event)
	}
	return events, nil
}
