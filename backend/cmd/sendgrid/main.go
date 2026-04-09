package main

import (
	"fmt"
	"log"

	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

func main() {
	client := sendgrid.NewSendClient("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")

	from := mail.NewEmail("Grapple Notifications", "support@grapplemma.com")
	subject := "Grapple MMA - Send Grid!"
	to := mail.NewEmail("Grapple Student", "jleevinn@gmail.com")
	plainTextContent := "easy to do anywhere, even with Go"
	htmlContent := "<strong>easy to do anywhere, even with Go</strong>"
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)

	// Send email
	response, err := client.Send(message)
	if err != nil {
		log.Println(err)
	} else {
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}

}
