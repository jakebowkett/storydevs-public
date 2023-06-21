package storydevs

import (
	"strings"
)

/*
Calendar widgets submitted with this number or greater will
be interpreted as being the current date. Dates are stored
as Unix timestamps with a precision of seconds. The below
threshold of 9 quadrillion seconds (approx 285388127 years)
was chosen because JavaScript's maximum safe integer cannot
fully accomodate the Go/Postgres max int64 value.
*/
const PresentThreshold = 9000000000000000

type Feedback map[string][]string

func (fb Feedback) Add(key, msg string) {
	msg = strings.ToUpper(string(msg[0])) + msg[1:]
	if !strings.HasSuffix(msg, ".") {
		msg += "."
	}
	fb[key] = append(fb[key], msg)
}

type Chain interface {
	Push(string)
	Pop()
	Current() string
	Len() int
}

type chain struct {
	slice []string
}

func NewChain() Chain {
	return &chain{}
}
func (c *chain) Push(s string) {
	c.slice = append(c.slice, s)
}
func (c *chain) Pop() {
	c.slice = c.slice[0 : len(c.slice)-1]
}
func (c chain) Current() string {
	return strings.ToLower(strings.Join(c.slice, "."))
}
func (c chain) Len() int {
	return len(c.slice)
}
