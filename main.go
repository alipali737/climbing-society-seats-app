package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/glebarez/go-sqlite"
)

type RegistrationData struct {
	Name    string `json:"name"`
	Member  bool   `json:"member"`
	EventID int    `json:"event"`
}

func main() {
	router := gin.Default()

	router.Static("/register", "./register")

	// API Endpoints
	router.GET("/", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, "/register") })
	router.GET("/api/events", handleEventDetails)
	router.POST("/api/register", handleAPIRegister)

	router.Run(":5500")
}

func consoleError(msg string) {
	fmt.Println("\033[31m[ERR] " + msg + "\033[0m")
}

func handleEventDetails(c *gin.Context) {
	// Get event_id from URL params
	eventIDParam := c.Query("event")
	if eventIDParam == "NaN" {
		return
	}

	eventID, err := strconv.Atoi(eventIDParam)
	if err != nil {
		msg := fmt.Sprintf("Failed to get event: %s", err)
		consoleError(msg)
		sendVisualResponse(c, false, msg, http.StatusNotFound)
		return
	}

	event, err := getEventByID(eventID)
	if err != nil {
		msg := fmt.Sprintf("Failed to get event: %s", err)
		consoleError(msg)
		sendVisualResponse(c, false, msg, http.StatusNotFound)
		return
	}

	response := gin.H{
		"session_location": event.EventLocation,
		"session_date":     event.EventDate,
		"meet_time":        event.MeetTime,
		"meet_point":       event.MeetLocation,
		"current_seats":    event.TotalSeats - event.SeatsTaken,
		"total_seats":      event.TotalSeats,
		"close_date":       event.CloseDatetime,
	}

	c.JSON(http.StatusOK, response)
}

type Event struct {
	EventID       int    `db:"event_id"`
	EventLocation string `db:"event_location"`
	EventDate     string `db:"event_date"`
	MeetLocation  string `db:"meet_location"`
	MeetTime      string `db:"meet_time"`
	TotalSeats    int    `db:"total_seats"`
	SeatsTaken    int    `db:"seats_taken"`
	RequireMember bool   `db:"require_member"`
	OpenDatetime  string `db:"open_datetime"`
	CloseDatetime string `db:"close_datetime"`
}

func (e *Event) addParticipant(firstName string, surname string, member bool) error {
	// Confirm there is space
	if e.TotalSeats-e.SeatsTaken <= 0 {
		return errors.New("no seats available")
	}

	// Create DB connection
	db, err := sql.Open("sqlite", "./database.db")
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

func getEventByID(eventID int) (*Event, error) {
	db, err := sql.Open("sqlite", "./database.db")
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := "SELECT event_id, event_location, event_date, meet_location, meet_time, total_seats, seats_taken, require_member, open_datetime, close_datetime FROM events WHERE event_id = ?"
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
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.New("event not found")
		}
		return nil, err
	}

	return &event, nil
}

func handleAPIRegister(c *gin.Context) {
	// // TODO: remove this once implemented
	// sendVisualResponse(c, false, "Functionality not implemented", http.StatusNotImplemented)
	// return

	// Process user data
	var registrationData RegistrationData
	if err := c.BindJSON(&registrationData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		consoleError(err.Error())
		return
	}

	// Process received data
	if !validateName(registrationData.Name) {
		msg := "Invalid name, please enter your first and last name"
		sendVisualResponse(c, false, msg, http.StatusBadRequest)
		consoleError(msg)
		return
	}

	// Get event details
	event, err := getEventByID(registrationData.EventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error})
		consoleError(err.Error())
		return
	}

	// Check that sign ups are open
	dateFormat := "02/01/2006 15:04:05"
	openTime, err := time.Parse(dateFormat, event.OpenDatetime)
	if err != nil {
		msg := "Unable to parse event open date"
		sendVisualResponse(c, false, msg, http.StatusInternalServerError)
		consoleError(msg)
		return
	}

	closeTime, err := time.Parse(dateFormat, event.CloseDatetime)
	if err != nil {
		msg := "Unable to parse event close date"
		sendVisualResponse(c, false, msg, http.StatusInternalServerError)
		consoleError(msg)
		return
	}

	currentTime := time.Now()
	if !(currentTime.After(openTime) && currentTime.Before(closeTime)) {
		msg := "The event is not currently open for registration"
		sendVisualResponse(c, false, msg, http.StatusForbidden)
		consoleError(msg)
		return
	}

	if event.RequireMember && !registrationData.Member {
		msg := "This event requires you to have paid membership fees"
		sendVisualResponse(c, false, msg, http.StatusForbidden)
		consoleError(msg)
		return
	}

	// Check that a seat is available
	if event.TotalSeats-event.SeatsTaken <= 0 {
		msg := "There are no seats available for this event"
		sendVisualResponse(c, false, msg, http.StatusForbidden)
		consoleError(msg)
		return
	}

	// Make updates to database
	formattedName := strings.Title(registrationData.Name)
	firstName, surname := splitName(formattedName)
	err = event.addParticipant(firstName, surname, registrationData.Member)
	if err != nil {
		msg := fmt.Sprintf("Failed to update database: %v", err)
		sendVisualResponse(c, false, msg, http.StatusInternalServerError)
		consoleError(msg)
		return
	}

	// Handle POST request
	sendVisualResponse(c, true, "You have been added to the event!", http.StatusOK)
}

func splitName(name string) (string, string) {
	parts := strings.Split(name, " ")
	firstName := parts[0]
	surname := strings.Join(parts[1:], " ")
	return firstName, surname
}

func validateName(name string) bool {
	name = strings.ToLower(name)
	regex := regexp.MustCompile("^[a-z]+ [a-z]+(-[a-z]+)*$")
	return regex.MatchString(name)
}

func sendVisualResponse(c *gin.Context, success bool, message string, statusCode int) {
	response := gin.H{
		"success": success,
		"message": message,
	}

	// Send JSON response
	c.JSON(statusCode, response)
}
