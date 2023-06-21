package setup

import (
	"github.com/jakebowkett/go-emailer/emailer"
	sd "github.com/jakebowkett/storydevs"
)

func mustEmailer(c *sd.Config) sd.Emailer {
	em := &emailer.Emailer{
		Host:     c.Credentials.EmailHost,
		Port:     c.Credentials.EmailPort,
		User:     c.Credentials.EmailUser,
		Pass:     c.Credentials.EmailPass,
		From:     c.Credentials.EmailFrom,
		Name:     c.Credentials.EmailName,
		Timeout:  c.EmailTimeout,
		Disabled: !c.EmailEnabled,
	}
	return em
}
