package mailchimp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type Client struct {
	http.Client
	apiHost string
	apiKey  string
}

type To struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
	Type  string `json:"name,omitempty"`
}

type Message struct {
	FromName      string   `json:"from_name"`
	FromEmail     string   `json:"from_email"`
	To            []To     `json:"to"`
	Subject       string   `json:"subject"`
	Text          string   `json:"text"`
	HTML          string   `json:"html"`
	SigningDomain string   `json:"signing_domain"`
	Tags          []string `json:"tags"`
	Subaccount    string   `json:"subaccount"`
	Metadata      struct {
		Website string
	} `json:"metadata"`
}

func New(host, key string) *Client {
	return &Client{
		apiHost: host,
		apiKey:  key,
	}
}

func (c *Client) SendEmail(ctx context.Context, subject, body, to string) error {
	url := fmt.Sprintf("%s/messages/send", c.apiHost)

	req := struct {
		Key     string  `json:"key"`
		Message Message `json:"message"`
	}{
		Key: c.apiKey,
		Message: Message{
			To: []To{
				{
					Email: to,
				},
			},
			FromEmail:     "support@grapplemma.com",
			FromName:      "Grapple Notifications",
			Subject:       subject,
			Text:          body,
			Subaccount:    "Grapple",
			SigningDomain: "grapplemma.com",
		},
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return err
	}
	fmt.Printf("Sending request:\n%v\n\n", string(reqBytes))

	resp, err := c.Post(url, "application/json", bytes.NewReader(reqBytes))
	if err != nil {
		return err
	}

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	log.Printf("Sent email:\n%v\n", string(respBytes))
	log.Printf("Resp code: %v\n", resp.StatusCode)

	return nil
}
