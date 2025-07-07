package mailer

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type MailTrapMailer struct {
	apiKey    string
	fromEmail string
}

func NewMailTrapMailer(apiKey, fromEmail string) (*MailTrapMailer, error) {
	if apiKey == "" {
		return &MailTrapMailer{}, errors.New("api key is required")
	}

	return &MailTrapMailer{
		apiKey:    apiKey,
		fromEmail: fromEmail,
	}, nil
}

func (m *MailTrapMailer) Send(templateFile, username, email string, data any, isSandbox bool) error {

	return nil
}

func main() {

	url := "https://sandbox.api.mailtrap.io/api/send/3872073"
	method := "POST"

	payload := strings.NewReader(`{\"from\":{\"email\":\"hello@example.com\",\"name\":\"Mailtrap Test\"},\"to\":[{\"email\":\"drazenpetrovic66@gmail.com\"}],\"subject\":\"You are awesome!\",\"text\":\"Congrats for sending test email with Mailtrap!\",\"category\":\"Integration Test\"}`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Authorization", "Bearer <YOUR_API_TOKEN>")
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(body))
}
