package emailer

import (
	"fmt"
	"net/smtp"
	"os"
)

func getEmailLoginCredentials() (email, password string) {
	return os.Getenv("SENDER_EMAIL"), os.Getenv("SENDER_PASSWORD")
}

func SendEmail(address, subject, message string) error {
	from, password := getEmailLoginCredentials()

	to := []string{
		address,
	}

	smtpHost := "smtp.gmail.com"
	smtpPort := "587"

	emailMessage := fmt.Sprintf("Subject: %s\r\n\r\n%s", subject, message)
	messageBytes := []byte(emailMessage)

	auth := smtp.PlainAuth("", from, password, smtpHost)

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, messageBytes)
	if err != nil {
		return err
	}

	return nil
}
