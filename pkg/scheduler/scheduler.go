package scheduler

import (
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/alipali737/climbing-society-seats-app/climbing-society-seats-app/pkg/database"
	"github.com/alipali737/climbing-society-seats-app/climbing-society-seats-app/pkg/emailer"
	"github.com/robfig/cron/v3"
)

func InitialiseScheduler() {
	c := cron.New()

	_, err := c.AddFunc("0 0 8 * * *", func() {
		checkScheduledEvents()
	})
	if err != nil {
		log.Println("Error scheduling function: ", err)
	}

	_, err = c.AddFunc("@every 1m", func() {
		checkClosedEvents()
	})
	if err != nil {
		log.Println("Error scheduling function: ", err)
	}

	c.Start()
}

func checkScheduledEvents() {
	events, err := database.GetEvents()
	if err != nil {
		log.Println(err)
	}

	var message string
	messageFormat := "## Post Event To %s Group At %v ##\nClimbing Session Signup!\nClimbing Location: %s\nMeet Time: %s\nMeet Location: %s\nSignups Close at: %s\n\n%s\n\n"

	// Iterate through each event
	for _, event := range events {
		currentDatetime, committeeMsgDatetime, openDatetime, _, err := getRoundedTimes(event)
		if err != nil {
			log.Println(err)
		}

		fmt.Printf("Current: %v; Committee: %v; Open: %v\n", currentDatetime, committeeMsgDatetime, openDatetime)

		if dateEqual(committeeMsgDatetime, currentDatetime) {
			message += fmt.Sprintf(messageFormat, "Committee", committeeMsgDatetime.String(), event.EventLocation, event.MeetTime, event.MeetLocation, event.CloseDatetime, event.GetLink())
		}

		if dateEqual(openDatetime, currentDatetime) {
			message += fmt.Sprintf(messageFormat, "Committee", committeeMsgDatetime.String(), event.EventLocation, event.MeetTime, event.MeetLocation, event.CloseDatetime, event.GetLink())
		}
	}

	fmt.Printf("Message to email: \n%s\n", message)

	if message != "" {
		fmt.Println("Sending Email")
		emailer.SendEmail(os.Getenv("EVENT_POSTS_EMAIL_ADDRESS"), "Society Session Event Posts to Send Today!", message)
	}
}

func checkClosedEvents() {
	events, err := database.GetEvents()
	if err != nil {
		log.Println(err)
	}

	for _, event := range events {
		if event.EventStatus == database.EventStatusClosed {
			continue
		}
		currentDatetime, _, _, closeDatetime, err := getRoundedTimes(event)
		if err != nil {
			log.Println(err)
		}
		if currentDatetime.After(closeDatetime) {
			type details struct {
				Event        database.Event
				Participants []database.Participant
			}

			participants, err := event.GetParticipants()
			if err != nil {
				log.Println(err)
			}

			data := details{
				Event:        event,
				Participants: participants,
			}

			msgTmpl, err := template.New("messageTemplate").Parse(EventOutputTemplate)
			if err != nil {
				log.Println(err)
			}
			output := &strings.Builder{}
			err = msgTmpl.Execute(output, data)

			message := output.String()
			fmt.Printf("Message to email: \n%s\n", message)

			if message != "" {
				fmt.Println("Sending Email")
				emailer.SendEmail(os.Getenv("EVENT_CLOSURE_EMAIL_ADDRESS"), fmt.Sprintf("Society Session Event %d Closed Today!", event.EventID), message)
			}

			err = event.Close()
			if err != nil {
				log.Println(err)
			}

		}
	}
}

func getRoundedTimes(event database.Event) (currentTime, committeeMsgTime, openTime, closeTime time.Time, err error) {
	// Check that sign ups are open
	dateFormat := "02/01/2006 15:04:05"
	openTime, err = time.Parse(dateFormat, event.OpenDatetime)
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, time.Time{}, err
	}
	openTime = openTime.Round(time.Minute)

	closeTime, err = time.Parse(dateFormat, event.CloseDatetime)
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, time.Time{}, err
	}
	closeTime = closeTime.Round(time.Minute)

	// Get current time in local timezone
	localTime := time.Now()

	// Get current time without timezone information for comparison
	currentTime = time.Date(localTime.Year(), localTime.Month(), localTime.Day(), localTime.Hour(), localTime.Minute(),
		localTime.Second(), localTime.Nanosecond(), time.UTC)
	currentTime = currentTime.Round(time.Minute)

	committeeMsgTime = openTime.Add(-(time.Hour * 24))

	return currentTime, committeeMsgTime, openTime, closeTime, nil
}

func dateEqual(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
