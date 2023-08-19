package run

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/alipali737/climbing-society-seats-app/climbing-society-seats-app/pkg/database"
	"github.com/alipali737/climbing-society-seats-app/climbing-society-seats-app/pkg/token"
	"github.com/gin-gonic/gin"
	_ "github.com/glebarez/go-sqlite"
)

type RegistrationData struct {
	Name    string `json:"name"`
	Member  bool   `json:"member"`
	EventID int    `json:"event"`
}

type Run struct{}

var encryptionPassPhrase string

func (r *Run) Run() error {
	generatedPassphrase, err := token.GenerateRandomPassphrase(32)
	if err != nil {
		return fmt.Errorf("failed to generate secure passphrase for encryption: %v", err)
	}
	encryptionPassPhrase = generatedPassphrase

	err = token.GenerateNewEncryptedToken(encryptionPassPhrase, 32)
	if err != nil {
		return fmt.Errorf("failed to generate new encrypted token: %v", err)
	}

	router := gin.Default()

	router.Static("/resources", "./resources")
	router.Static("/register", "./register")

	router.GET("/admin", func(c *gin.Context) {
		c.File("./admin/index.html")
	})
	router.GET("/admin/dashboard", authMiddleware(encryptionPassPhrase), func(c *gin.Context) {
		c.File("./admin/dashboard.html")
	})

	// API Endpoints
	router.GET("/", func(c *gin.Context) { c.Redirect(http.StatusMovedPermanently, "/register") })

	router.POST("/api/register", handleAPIRegister)

	router.POST("/api/login", handleAdminLogin)

	router.GET("/api/event", handleEventDetails)
	router.DELETE("/api/event", authMiddleware(encryptionPassPhrase), handleDeleteEvent)

	router.GET("/api/events", authMiddleware(encryptionPassPhrase), handleGetEvents)
	router.POST("/api/events", authMiddleware(encryptionPassPhrase), handleCreateEvent)
	router.PUT("/api/events", authMiddleware(encryptionPassPhrase), handleUpdateEvent)

	router.GET("/api/participants", authMiddleware(encryptionPassPhrase), handleGetEventParticipants)
	router.DELETE("/api/participant", authMiddleware(encryptionPassPhrase), handleDeleteParticipant)

	router.Run(":443")
	return nil
}

func authMiddleware(encryptionPassPhrase string) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie("token")
		if err != nil {
			// Return a 404, hide the existence of the page if they are not authorized to view it
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
			consoleError("No auth token present")
			c.Abort()
			return
		}

		key, err := token.GetCryptographicKey(encryptionPassPhrase)
		if err != nil {
			// Return a 404, hide the existence of the page if they are not authorized to view it
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
			consoleError("No auth token present")
			c.Abort()
			return
		}

		token, err := token.ValidateJWT(tokenString, key)
		if err != nil {
			// Return a 404, hide the existence of the page if they are not authorized to view it
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
			consoleError(fmt.Sprintf("Auth failed: %v\n", err))
			c.Abort()
			return
		}

		consoleLog("Authenticated token, proceeding")

		c.Set("claims", token.Claims)
		c.Next()
	}
}

func consoleError(msg string) {
	fmt.Println("\033[31m[ERR] " + msg + "\033[0m")
}

func consoleLog(msg string) {
	fmt.Println("[DBG] " + msg)
}

func handleUpdateEvent(c *gin.Context) {
	// Get event_id from URL params
	eventIDParam := c.Query("event")
	if eventIDParam == "NaN" {
		return
	}

	eventID, err := strconv.Atoi(eventIDParam)
	if err != nil {
		msg := fmt.Sprintf("Failed to update event: %s", err)
		consoleError(msg)
		sendResponse(c, false, msg, http.StatusNotFound)
		return
	}

	var event Event
	if err := c.BindJSON(&event); err != nil {
		msg := fmt.Sprintf("Failed to update event: %s", err)
		consoleError(msg)
		sendResponse(c, false, msg, http.StatusBadRequest)
		return
	}

	if err := updateEventInDatabase(eventID, event); err != nil {
		msg := fmt.Sprintf("Failed to update event: %s", err)
		consoleError(msg)
		sendResponse(c, false, msg, http.StatusInternalServerError)
		return
	}

	sendResponse(c, true, "Successfully updated event", http.StatusOK)
}

func updateEventInDatabase(eventID int, eventData Event) error {
	db, err := sql.Open("sqlite", "./database.db")
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
            require_member = ?,
            open_datetime = ?,
            close_datetime = ?
        WHERE event_id = ?
    `

	_, err = db.Exec(
		query,
		eventData.EventLocation,
		eventData.EventDate,
		eventData.MeetLocation,
		eventData.MeetTime,
		eventData.TotalSeats,
		eventData.RequireMember,
		eventData.OpenDatetime,
		eventData.CloseDatetime,
		eventID,
	)
	if err != nil {
		return err
	}

	return nil
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
		sendResponse(c, false, msg, http.StatusNotFound)
		return
	}

	event, err := getEventByID(eventID)
	if err != nil {
		msg := fmt.Sprintf("Failed to get event: %s", err)
		consoleError(msg)
		sendResponse(c, false, msg, http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, event)
}

func handleDeleteEvent(c *gin.Context) {
	// Get event from URL params
	eventIDParam := c.Query("event")
	if eventIDParam == "NaN" {
		return
	}

	eventID, err := strconv.Atoi(eventIDParam)
	if err != nil {
		msg := fmt.Sprintf("Failed to find event: %s", err)
		consoleError(msg)
		sendResponse(c, false, msg, http.StatusNotFound)
		return
	}

	err = deleteEvent(eventID)
	if err != nil {
		msg := fmt.Sprintf("Failed to delete event: %s", err)
		consoleError(msg)
		sendResponse(c, false, msg, http.StatusInternalServerError)
		return
	}

	sendResponse(c, true, "Successfully deleted event", http.StatusOK)
}

func handleCreateEvent(c *gin.Context) {
	var event Event
	if err := c.BindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		consoleError(err.Error())
		return
	}

	err := createEvent(event)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		consoleError(err.Error())
		return
	}

	// Handle POST request
	sendResponse(c, true, "Event added!", http.StatusOK)
}

func createEvent(event Event) error {
	db, err := sql.Open("sqlite", "./database.db")
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

func deleteEvent(eventId int) error {
	db, err := sql.Open("sqlite", "./database.db")
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

func handleDeleteParticipant(c *gin.Context) {
	// Get participant from URL params
	participantIDParam := c.Query("participant")
	if participantIDParam == "NaN" {
		return
	}

	participantID, err := strconv.Atoi(participantIDParam)
	if err != nil {
		msg := fmt.Sprintf("Failed to find participant: %s", err)
		consoleError(msg)
		sendResponse(c, false, msg, http.StatusNotFound)
		return
	}

	err = deleteParticipant(participantID)
	if err != nil {
		msg := fmt.Sprintf("Failed to delete participant: %s", err)
		consoleError(msg)
		sendResponse(c, false, msg, http.StatusInternalServerError)
		return
	}

	sendResponse(c, true, "Successfully deleted participant", http.StatusOK)
}

func deleteParticipant(participantID int) error {
	db, err := sql.Open("sqlite", "./database.db")
	if err != nil {
		return err
	}
	defer db.Close()

	query := "DELETE FROM participants WHERE participant_id = ?"
	res, err := db.Exec(query, participantID)
	fmt.Println(res)
	if err != nil {
		return err
	}

	return nil
}

func handleGetEvents(c *gin.Context) {
	db, err := sql.Open("sqlite", "./database.db")
	if err != nil {
		msg := fmt.Sprintf("Failed to connect to database: %s", err)
		consoleError(msg)
		sendResponse(c, false, msg, http.StatusInternalServerError)
		return
	}
	defer db.Close()

	query := "SELECT event_id, event_location, event_date, meet_location, meet_time, total_seats, seats_taken, require_member, open_datetime, close_datetime FROM events"
	rows, err := db.Query(query)
	if err != nil {
		msg := fmt.Sprintf("Failed to get events: %s", err)
		consoleError(msg)
		sendResponse(c, false, msg, http.StatusInternalServerError)
		return
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
		)
		if err != nil {
			msg := fmt.Sprintf("Failed to parse event: %s", err)
			consoleError(msg)
			sendResponse(c, false, msg, http.StatusInternalServerError)
			return
		}
		events = append(events, event)
	}

	c.JSON(http.StatusOK, events)
}

func handleGetEventParticipants(c *gin.Context) {
	// Get event_id from URL params
	eventIDParam := c.Query("event")
	if eventIDParam == "NaN" {
		return
	}

	eventID, err := strconv.Atoi(eventIDParam)
	if err != nil {
		msg := fmt.Sprintf("Failed to get event participants: %s", err)
		consoleError(msg)
		sendResponse(c, false, msg, http.StatusNotFound)
		return
	}

	participants, err := getEventParticipants(eventID)
	if err != nil {
		msg := fmt.Sprintf("Failed to get event participants: %s", err)
		consoleError(msg)
		sendResponse(c, false, msg, http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, participants)
}

type Event struct {
	EventID       int    `db:"event_id" json:"event_id"`
	EventLocation string `db:"event_location" json:"session_location"`
	EventDate     string `db:"event_date" json:"session_date"`
	MeetLocation  string `db:"meet_location" json:"meet_point"`
	MeetTime      string `db:"meet_time" json:"meet_time"`
	TotalSeats    int    `db:"total_seats" json:"total_seats"`
	SeatsTaken    int    `db:"seats_taken" json:"current_seats"`
	RequireMember bool   `db:"require_member" json:"require_member"`
	OpenDatetime  string `db:"open_datetime" json:"open_date"`
	CloseDatetime string `db:"close_datetime" json:"close_date"`
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

type Participant struct {
	ParticipantID int    `db:"participant_id" json:"participant_id"`
	EventID       int    `db:"event_id" json:"event_id"`
	FirstName     string `db:"first_name" json:"first_name"`
	LastName      string `db:"surname" json:"last_name"`
	Member        bool   `db:"member" json:"member"`
}

func getEventParticipants(eventID int) ([]Participant, error) {
	db, err := sql.Open("sqlite", "./database.db")
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

func handleAPIRegister(c *gin.Context) {
	// Process user data
	var registrationData RegistrationData
	if err := c.ShouldBindJSON(&registrationData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		consoleError(err.Error())
		return
	}

	// Process received data
	if !validateName(registrationData.Name) {
		msg := "Invalid name, please enter your first and last name"
		sendResponse(c, false, msg, http.StatusBadRequest)
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
		sendResponse(c, false, msg, http.StatusInternalServerError)
		consoleError(msg)
		return
	}

	closeTime, err := time.Parse(dateFormat, event.CloseDatetime)
	if err != nil {
		msg := "Unable to parse event close date"
		sendResponse(c, false, msg, http.StatusInternalServerError)
		consoleError(msg)
		return
	}

	// Get current time in local timezone
	localTime := time.Now()

	// Get current time without timezone information for comparison
	currentTime := time.Date(localTime.Year(), localTime.Month(), localTime.Day(), localTime.Hour(), localTime.Minute(),
		localTime.Second(), localTime.Nanosecond(), time.UTC)

	if !(currentTime.After(openTime) && currentTime.Before(closeTime)) {
		msg := "The event is not currently open for registration"
		sendResponse(c, false, msg, http.StatusForbidden)
		consoleError(msg)
		return
	}

	if event.RequireMember && !registrationData.Member {
		msg := "This event requires you to have paid membership fees"
		sendResponse(c, false, msg, http.StatusForbidden)
		consoleError(msg)
		return
	}

	// Check that a seat is available
	if event.TotalSeats-event.SeatsTaken <= 0 {
		msg := "There are no seats available for this event"
		sendResponse(c, false, msg, http.StatusForbidden)
		consoleError(msg)
		return
	}

	// Make updates to database
	formattedName := strings.Title(registrationData.Name)
	firstName, surname := splitName(formattedName)
	err = event.addParticipant(firstName, surname, registrationData.Member)
	if err != nil {
		msg := fmt.Sprintf("Failed to update database: %v", err)
		sendResponse(c, false, msg, http.StatusInternalServerError)
		consoleError(msg)
		return
	}

	// Handle POST request
	sendResponse(c, true, "You have been added to the event!", http.StatusOK)
}

func handleAdminLogin(c *gin.Context) {
	type LoginData struct {
		Username string
		Password string
	}

	// Process user data
	var loginData LoginData
	if err := c.BindJSON(&loginData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		consoleError(err.Error())
		return
	}

	// Validate credentials
	dbUser, err := database.GetUserFromDatabaseByUsername(loginData.Username)
	if err != nil {
		response := gin.H{
			"success": false,
			"message": "Invalid username or password",
		}

		// Send JSON response
		c.JSON(http.StatusForbidden, response)
		return
	}

	if database.ValidatePassword(loginData.Password, dbUser.PasswordHash) {
		key, err := token.GetCryptographicKey(encryptionPassPhrase)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			consoleError(err.Error())
			return
		}

		token, err := token.NewJWT(loginData.Username, key)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			consoleError(err.Error())
			return
		}

		response := gin.H{
			"success": true,
			"message": "Authentication successful",
		}

		c.SetCookie("token", token, 3600, "/", "", false, true)

		// Send JSON response
		c.JSON(http.StatusAccepted, response)
		return
	} else {
		response := gin.H{
			"success": false,
			"message": "Invalid username or password",
		}

		// Send JSON response
		c.JSON(http.StatusForbidden, response)
		return
	}
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

func sendResponse(c *gin.Context, success bool, message string, statusCode int) {
	response := gin.H{
		"success": success,
		"message": message,
	}

	// Send JSON response
	c.JSON(statusCode, response)
}
