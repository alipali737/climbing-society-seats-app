package scheduler

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alipali737/climbing-society-seats-app/climbing-society-seats-app/pkg/database"
	"github.com/alipali737/climbing-society-seats-app/climbing-society-seats-app/pkg/emailer"
	"github.com/robfig/cron/v3"
)

func InitialiseScheduler() {
	c := cron.New()

	_, err := c.AddFunc("0 0 8 * * *", func() {
		checkEvents()
	})
	if err != nil {
		log.Fatal("Error scheduling function: ", err)
	}

	c.Start()
}

func checkEvents() {
	events, err := database.GetEvents()
	if err != nil {
		log.Fatal(err)
	}

	var message string
	messageFormat := "## Post Event To %s Group At %v ##\nClimbing Session Signup!\nClimbing Location: %s\nMeet Time: %s\nMeet Location: %s\nSignups Close at: %s\n\n%s\n\n"

	// Iterate through each event
	for _, event := range events {
		currentDatetime, committeeMsgDatetime, openDatetime, err := getRoundedTimes(event)
		if err != nil {
			log.Fatal(err)
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

func getRoundedTimes(event database.Event) (currentTime, committeeMsgTime, openTime time.Time, err error) {
	// Check that sign ups are open
	dateFormat := "02/01/2006 15:04:05"
	openTime, err = time.Parse(dateFormat, event.OpenDatetime)
	if err != nil {
		return time.Time{}, time.Time{}, time.Time{}, err
	}
	openTime = openTime.Round(time.Minute)

	// Get current time in local timezone
	localTime := time.Now()

	// Get current time without timezone information for comparison
	currentTime = time.Date(localTime.Year(), localTime.Month(), localTime.Day(), localTime.Hour(), localTime.Minute(),
		localTime.Second(), localTime.Nanosecond(), time.UTC)
	currentTime = currentTime.Round(time.Minute)

	committeeMsgTime = openTime.Add(-(time.Hour * 24))

	return currentTime, committeeMsgTime, openTime, nil
}

func dateEqual(date1, date2 time.Time) bool {
	y1, m1, d1 := date1.Date()
	y2, m2, d2 := date2.Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
