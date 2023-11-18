package run

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/alipali737/climbing-society-seats-app/climbing-society-seats-app/pkg/database"
	"github.com/alipali737/climbing-society-seats-app/climbing-society-seats-app/pkg/scheduler"
	"github.com/alipali737/climbing-society-seats-app/climbing-society-seats-app/pkg/token"
	"github.com/gin-gonic/gin"
	_ "github.com/glebarez/go-sqlite"
	"github.com/joho/godotenv"
)

type RegistrationData struct {
	Name    string `json:"name"`
	Member  bool   `json:"member"`
	EventID int    `json:"event"`
}

type Run struct{}

var encryptionPassPhrase string

func (r *Run) Run() error {
	err := godotenv.Load("./config.env")
	if err != nil {
		return err
	}
	generatedPassphrase, err := token.GenerateRandomPassphrase(32)
	if err != nil {
		return fmt.Errorf("failed to generate secure passphrase for encryption: %v", err)
	}
	encryptionPassPhrase = generatedPassphrase

	err = token.GenerateNewEncryptedToken(encryptionPassPhrase, 32)
	if err != nil {
		return fmt.Errorf("failed to generate new encrypted token: %v", err)
	}

	scheduler.CheckScheduledEvents()

	scheduler.InitialiseScheduler()

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

	router.Run(":8080")
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

	var event database.Event
	if err := c.BindJSON(&event); err != nil {
		msg := fmt.Sprintf("Failed to update event: %s", err)
		consoleError(msg)
		sendResponse(c, false, msg, http.StatusBadRequest)
		return
	}

	eventParticipants, err := database.GetEventParticipants(event.EventID)
	if err != nil {
		msg := fmt.Sprintf("Failed to get event participants for update: %s", err)
		consoleError(msg)
		sendResponse(c, false, msg, http.StatusBadRequest)
		return
	}

	event.SeatsTaken = len(eventParticipants)

	oldEvent, err := database.GetEventByID(event.EventID)
	if err != nil {
		msg := fmt.Sprintf("Failed to get old event for update: %s", err)
		consoleError(msg)
		sendResponse(c, false, msg, http.StatusInternalServerError)
		return
	}

	event.EventStatus = oldEvent.EventStatus

	if err := database.UpdateEventInDatabase(eventID, event); err != nil {
		msg := fmt.Sprintf("Failed to update event: %s", err)
		consoleError(msg)
		sendResponse(c, false, msg, http.StatusInternalServerError)
		return
	}

	sendResponse(c, true, "Successfully updated event", http.StatusOK)
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

	event, err := database.GetEventByID(eventID)
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

	err = database.DeleteEvent(eventID)
	if err != nil {
		msg := fmt.Sprintf("Failed to delete event: %s", err)
		consoleError(msg)
		sendResponse(c, false, msg, http.StatusInternalServerError)
		return
	}

	sendResponse(c, true, "Successfully deleted event", http.StatusOK)
}

func handleCreateEvent(c *gin.Context) {
	var event database.Event
	if err := c.BindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		consoleError(err.Error())
		return
	}

	err := database.CreateEvent(event)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		consoleError(err.Error())
		return
	}

	scheduler.CheckScheduledEvents()

	// Handle POST request
	sendResponse(c, true, "Event added!", http.StatusOK)
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

	err = database.DeleteParticipant(participantID)
	if err != nil {
		msg := fmt.Sprintf("Failed to delete participant: %s", err)
		consoleError(msg)
		sendResponse(c, false, msg, http.StatusInternalServerError)
		return
	}

	sendResponse(c, true, "Successfully deleted participant", http.StatusOK)
}

func handleGetEvents(c *gin.Context) {
	events, err := database.GetEvents()
	if err != nil {
		consoleError(err.Error())
		sendResponse(c, false, err.Error(), http.StatusInternalServerError)
		return
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

	participants, err := database.GetEventParticipants(eventID)
	if err != nil {
		msg := fmt.Sprintf("Failed to get event participants: %s", err)
		consoleError(msg)
		sendResponse(c, false, msg, http.StatusNotFound)
		return
	}

	c.JSON(http.StatusOK, participants)
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
	event, err := database.GetEventByID(registrationData.EventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error})
		consoleError(err.Error())
		return
	}

	// Check that sign ups are open
	dateFormat := "02/01/2006 15:04:05"

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

	if !(currentTime.Before(closeTime)) {
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
	err = event.AddParticipant(firstName, surname, registrationData.Member)
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
