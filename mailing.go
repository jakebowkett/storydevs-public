package storydevs

type Emailer interface {
	Send(recipient string, subject, body string) (err error)
}
