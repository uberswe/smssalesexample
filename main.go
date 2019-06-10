package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
)

type Message int

const (
	Positive Message = 0
	Negative Message = 1
	Neutral  Message = 2
)

type Customer struct {
	Replied     bool
	Done        bool
	MessageType Message
}

var (
	port = ":80"

	numberFrom   = ""
	username     = ""
	password     = ""
	slackWebhook = ""

	// A map of customers so we can handle multiple phone numbers/customers
	customers map[string]Customer

	message1        = "Hello Markus, thank you for visiting GoPHP.io! On a scale from 1 to 10, with 10 being the best, how would you rate your experience?"
	messagePositive = "We are happy you had a good experience at GoPHP.io! PLease reply back to use if you would like to tell us why you liked the expereince"
	messageNegative = "We are sorry GoPHP.io did not live up to your expectations. Please reply back to us if you would like to tell us why you were not satisfied"
	messageFinal    = "Thank you for your feedback, if you would like to get in touch with use please send an email to markus@gophp.io"
)

func main() {
	// We take all arguments after our program name
	args := os.Args[1:]

	// Error if not enough arguments
	if len(args) != 4 {
		panic("must supply 46elks username and password, from number and a Slack webhook")
	}

	username = args[0]
	password = args[1]
	numberFrom = args[2]
	slackWebhook = args[3]
	customers = make(map[string]Customer)

	// This sets up two endpoints http://localhost:8080/outgoing and http://localhost:8080/incoming
	// I recommend running this program behind something like Caddy (https://caddyserver.com/) that provides SSL and can proxy to the localhost
	http.HandleFunc("/outgoing", handleOutgoingSMS) // setting router rule
	http.HandleFunc("/incoming", handleIncomingSMS)
	fmt.Println("Listening on port " + port)
	err := http.ListenAndServe(port, nil) // setting listening port
	if err != nil {
		panic(err)
	}
}

// handleOutgoingSMS parses the field "to" from a post request and passes it to the sendSMS function
func handleOutgoingSMS(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		fmt.Println(err)
		return
	}

	to := r.FormValue("to")

	sendSMS(to, message1)

}

// handleIncomingSMS receives requests from the 46elks API when a new text message is received
func handleIncomingSMS(w http.ResponseWriter, r *http.Request) {
	var customer Customer
	// According to the documentation we can expect incoming messages to come in the following format as a post request in application/x-www-form-urlencoded
	//
	// direction=incoming&
	// id=sf8425555e5d8db61dda7a7b3f1b91bdb&
	// from=%2B46706861004&to=%2B46706861020&
	// created=2018-07-13T13%3A57%3A23.741000&
	// message=Hello%20how%20are%20you%3F

	if err := r.ParseForm(); err != nil {
		fmt.Println(err)
		return
	}

	// Direction is 'incoming'
	direction := r.FormValue("direction")
	// ID of the message
	id := r.FormValue("id")
	// Phone number message was sent from
	from := r.FormValue("from")
	// Our own number
	to := r.FormValue("to")
	// Date time when message was created
	created := r.FormValue("created")
	// The text message itself
	message := r.FormValue("message")

	fmt.Printf("[%s][%s][%s] %s => %s: %s\n", created, direction, id, from, to, message)

	// We check if the customer exists
	if val, ok := customers[from]; ok {
		customer = val
	} else {
		// If the customer does not exist it means we have not received a reply before
		customer = Customer{Replied: false, Done: false, MessageType: Neutral}
	}
	if !customer.Replied && !customer.Done {
		msgType := determineMessageType(message)
		// I would say an 8 or higher is positive, 7 or lower would indicate something is wrong (just my assumption)
		if msgType == Positive {
			sendSMS(from, messagePositive)
			customer.MessageType = msgType
			customer.Replied = true
			customers[from] = customer
			updateSlack(from, message, customer.MessageType)
			return
		} else if msgType == Negative {
			// If we did not have a message with 8 or higher
			sendSMS(from, messageNegative)
			customer.MessageType = msgType
			customer.Replied = true
			customers[from] = customer
			updateSlack(from, message, customer.MessageType)
			return
		}
		// If we are unable to determine a positive or negative answer
		// In a live environment I would recommend saving these longer responses and maybe sending it to a support desk to manually handle them
		sendSMS(from, messageFinal)
		customer.Replied = true
		customer.Done = true
		customers[from] = customer
		updateSlack(from, message, customer.MessageType)
		return

	} else if !customer.Done {
		sendSMS(from, messageFinal)
		customer.Done = true
		customers[from] = customer
		updateSlack(from, message, customer.MessageType)
		return
	}
}

// sendSMS sends a post request to 46elks which then tries to send a text message to the phone number provided as the "to" string
func sendSMS(to string, message string) {
	data := url.Values{
		"from":    {numberFrom},
		"to":      {to},
		"message": {message}}

	req, err := http.NewRequest("POST", "https://api.46elks.com/a1/SMS", bytes.NewBufferString(data.Encode()))
	if err != nil {
		fmt.Println(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	req.SetBasicAuth(username, password)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("Sent sms to %s: %s\n", to, message)
	fmt.Println(string(body))
}

func determineMessageType(message string) Message {
	// If the message is less than 2 characters in length we try to convert to int
	if len(message) <= 2 {
		value, err := strconv.Atoi(message)
		// If we successfully converted to an int we can process the message
		if err == nil {
			// I would say an 8 or higher is positive, 7 or lower would indicate something is wrong (just my assumption)
			if value >= 8 {
				return Positive
			} else if value <= 3 {
				// If we did not have a message with 8 or higher
				return Negative

			}
		}
	}
	// If we are unable to determine a positive or negative answer
	return Neutral
}

func updateSlack(from string, message string, t Message) {
	// I like to color code the replies, here we have red for negative, green for positive and gray for neutral
	color := "#C0C0C0"
	if t == Positive {
		color = "#006400"
	} else if t == Negative {
		color = "#8B0000"
	}
	var jsonStr = []byte(`{
			"attachments": [
				{
					"fallback": "` + message + `",
					"color": "` + color + `",
					"text": "` + message + `",
					"fields": [
						{
							"title": "Tel",
							"value": "` + from + `",
							"short": false
						}
					],
				}
    		]}`)
	// See the Slack documentation on creating webhooks https://api.slack.com/incoming-webhooks
	req, err := http.NewRequest("POST", slackWebhook, bytes.NewBuffer(jsonStr))
	if err != nil {
		fmt.Println(err)
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Content-Length", strconv.Itoa(len(jsonStr)))
	req.SetBasicAuth(username, password)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		fmt.Println(err)
	}

	fmt.Printf("Sent slack message: %s - %s\n", from, message)
	fmt.Println(string(body))
}
