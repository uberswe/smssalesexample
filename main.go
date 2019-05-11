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

var port = ":80"

var numberFrom = ""
var username = ""
var password = ""

// I use two bools since I only run this for one phone number, if you have multiple phone numbers I would change this to a map like map[phone]replied or something similar
var replied = false
var done = false

var message1 = "Hello Markus, thank you for visiting GoPHP.io! On a scale from 1 to 10, with 10 being the best, how would you rate your experience?"
var messagepositive = "Thank you for your feedback, we are happy you had a good experience at GoPHP.io! Please reply back to us if you would like to tell us why you liked the experience"
var messagenegative = "We are sorry that you did not enjoy visiting GoPHP.io. If there is anything we can do to make the experience better please do not hesitate to contact us at markus@gophp.io."
var messagefinal = "Thank you for your feedback, if you would like to get in touch with us please send an email to markus@gophp.io."

func main() {
	// We take all arguments after our program name
	args := os.Args[1:]

	// Error if not enough arguments
	if len(args) != 3 {
		panic("must supply 46elks username and password and from number")
	}

	username = args[0]
	password = args[1]
	numberFrom = args[2]

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

func handleOutgoingSMS(w http.ResponseWriter, r *http.Request) {

	if err := r.ParseForm(); err != nil {
		fmt.Println(err)
		return
	}

	// Direction is 'incoming'
	to := r.FormValue("to")

	sendSMS(to, message1)

}

func handleIncomingSMS(w http.ResponseWriter, r *http.Request) {
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

	if !replied && !done {
		// If the message is less than 2 characters in length we try to convert to int
		if len(message) <= 2 {
			value, err := strconv.Atoi(message)
			// If we successfully converted to an int we can process the message
			if err != nil {
				// I would say an 8 or higher is positive, 7 or lower would indicate something is wrong (just my assumption)
				if value >= 8 {
					sendSMS(to, messagepositive)
					replied = true
					return
				} else {
					// If we did not have a message with 8 or higher
					sendSMS(to, messagenegative)
					replied = true
					return
				}
			}
		}
		// If we are unable to determine a positive or negative answer
		// In a live environment I would recommend saving these longer responses and maybe sending it to a support desk to manually handle them
		sendSMS(to, messagefinal)
		replied = true
		done = true
		return

	} else if !done {
		sendSMS(to, messagefinal)
		done = true
		return
	}
}

func sendSMS(to string, message string) {
	data := url.Values{
		"from":    {numberFrom},
		"to":      {to},
		"message": {message}}

	req, err := http.NewRequest("POST", "https://api.46elks.com/a1/SMS", bytes.NewBufferString(data.Encode()))
	if err != nil {
		panic(err)
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", strconv.Itoa(len(data.Encode())))
	req.SetBasicAuth(username, password)

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		panic(err)
	}

	fmt.Printf("Sent sms to %s: %s\n", to, message1)
	fmt.Println(string(body))
}
