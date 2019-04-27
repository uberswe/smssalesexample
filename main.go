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

var numberFrom = ""
var username = ""
var password = ""

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
	fmt.Println("Listening on port :8080")
	err := http.ListenAndServe(":8080", nil) // setting listening port
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
	// ID of the message
	message := r.FormValue("message")

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

	fmt.Printf("Sent sms to %s: %s\n", to, message)
	fmt.Println(string(body))
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
}
