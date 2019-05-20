## SMS Sales Example

An example of a customer feedback bot to rate and get text input regarding past sales via text messaging written in go.

Run like so: `go run main.go [46elks username] [46elks password] [Elks46 phone number] [Slack webhook]`

You can register for a 46elks account on their [website](https://46elks.com).

Read about Slack webhooks and how to get your webhook url in the [Slack Incoming Webhook Documentation](https://api.slack.com/incoming-webhooks)

To send an email just send a post request with a `to` field to `/outgoing`, below is an example in curl

```
curl -X POST \
  http://<your url here>/outgoing \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d 'to=<your number here>'
```