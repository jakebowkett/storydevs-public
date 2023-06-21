package setup

import (
	"io/ioutil"
	"strings"

	"github.com/jakebowkett/go-hyphenate/hyphenate"
	sd "github.com/jakebowkett/storydevs"
)

func mustHyphenator(c *sd.Config) sd.Hyphenator {

	f, err := ioutil.ReadFile(c.PathHyphenateCustom)
	if err != nil {
		panic(err)
	}

	m := make(map[string][]string)
	lines := strings.Split(string(f), "\n")

	for _, line := range lines {

		ss := strings.Fields(line)
		if len(ss) != 2 {
			panic("expected 2 words per line")
		}

		m[ss[0]] = strings.Split(ss[1], "-")
	}

	hyphen := "­" // This is a shy hyphen, not an empty string.
	h, err := hyphenate.New(c.PathHyphenate, hyphen, m)
	if err != nil {
		panic(err)
	}

	return h
}
