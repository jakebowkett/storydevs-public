package setup

import (
	"fmt"
	"regexp"
	"sort"
	"time"

	sd "github.com/jakebowkett/storydevs"
)

func initTimezone(f *sd.Field) error {
	isOffset, err := regexp.Compile(`^(\+|-)\d+$`)
	if err != nil {
		return err
	}
	vv := f.Value
	skip := []string{
		"local",
		"utc",
	}
	for i := range vv {
		s := vv[i].Name
		if in(skip, s) {
			continue
		}
		loc, err := time.LoadLocation(s)
		if err != nil {
			return err
		}
		now := time.Now()
		now = now.In(loc)
		abbr, offset := now.Zone()
		if isOffset.MatchString(abbr) {
			abbr = ""
		}
		minutes := offset / 60
		hours := minutes / 60
		minutes -= hours * 60
		if minutes < 0 {
			minutes *= -1
		}
		vv[i].Value = append(vv[i].Value,
			sd.Value{
				Name: "abbr",
				Text: abbr,
			},
			sd.Value{
				Idx:  offset,
				Name: "offset",
				Text: fmt.Sprintf("UTC%+03d:%02d", hours, minutes),
			},
		)
	}
	sort.SliceStable(vv, func(i, j int) bool {
		if vv[i].Name == "local" {
			return true
		}
		if vv[j].Name == "local" {
			return false
		}
		if vv[i].Name == "utc" {
			return true
		}
		if vv[j].Name == "utc" {
			return false
		}
		return vv[i].Value[1].Idx < vv[j].Value[1].Idx
	})
	return nil
}
