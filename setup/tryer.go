package setup

import (
	"fmt"
	"time"

	"github.com/jakebowkett/go-retry/retry"
	sd "github.com/jakebowkett/storydevs"
	// "github.com/jakebowkett/storydevs/postgres"
)

func mustTryer(kind string, c *sd.Config, tryAgain retry.Retry) sd.Tryer {

	r, ok := c.Retry[kind]
	if !ok {
		panic(fmt.Errorf("config.Retry[%q] doesn't exist", kind))
	}

	// t, err := retry.New(postgres.Retry, retry.Options{
	t, err := retry.New(tryAgain, retry.Options{
		Retries:     r.Retries,
		Base:        time.Millisecond * time.Duration(r.Base),
		MaxInterval: time.Millisecond * time.Duration(r.MaxInterval),
		MaxWait:     time.Millisecond * time.Duration(r.MaxWait),
		Exponent:    r.Exponent,
		Jitter:      r.Jitter,
	})
	if err != nil {
		panic(err)
	}

	return t
}
