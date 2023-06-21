package setup

import (
	"fmt"
	"html/template"
	"math/rand"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/jakebowkett/go-num/num"
	"github.com/jakebowkett/go-view/view"
	sd "github.com/jakebowkett/storydevs"
)

var contiguousNewlines = regexp.MustCompile(`\n+`)

func templates(c *sd.Config, h sd.Hyphenator) sd.View {

	view := view.New(map[string]interface{}{
		"splitCalendar": func(s string) []string {

			if s == "Present" {
				return []string{"", "", ""}
			}
			if s == "" {
				return []string{"", "", ""}
			}
			ss := strings.Split(s, " ")

			// Order in yyyy/mm/dd format.
			if len(ss) == 2 {
				ss[0], ss[1] = ss[1], ss[0]
				ss = append(ss, "")
			} else {
				ss[0], ss[1], ss[2] = ss[2], ss[0], ss[1]
				ss[2] = strings.Trim(ss[2], ",")
			}

			/*
				Convert month name to number. We subtract 1 from
				the month number to match JavaScript's zero-month
				numbering.
			*/
			t, err := time.Parse("January", ss[1])
			if err != nil {
				panic(err)
			}
			ss[1] = strconv.Itoa(int(t.Month() - 1))
			return ss
		},
		"splitTime": func(s string) []string {
			if s == "" {
				return []string{"", "", ""}
			}
			n, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				panic(err)
			}
			return sd.TimeValue(n)
		},
		"strToParas": func(s string) []string {
			s = contiguousNewlines.ReplaceAllString(s, "\n")
			s = strings.TrimSpace(s)
			return strings.Split(s, "\n")
		},
		"in": func(ss []string, s string) bool {
			for i := range ss {
				if ss[i] == s {
					return true
				}
			}
			return false
		},
		"capitalise": func(s string) string {
			for _, r := range s {
				c := string(r)
				s = strings.ToUpper(c) + s[len(c):]
				break
			}
			return s
		},
		"join": func(ss ...string) string {
			return strings.Join(ss, "")
		},
		"spew": func(v interface{}) string {
			spew.Dump(v)
			return ""
		},
		"log": func(v interface{}) string {
			fmt.Println(v)
			return ""
		},
		"fileFromURL": func(url string) string {
			parts := strings.Split(url, "/")
			return parts[len(parts)-1]
		},
		"fileToThumb": func(fn string) string {
			if fn == "" {
				return fn
			}
			if strings.Contains(fn, "_thumb") {
				return fn
			}
			ext := filepath.Ext(fn)
			return strings.TrimSuffix(fn, ext) + "_thumb" + ext
		},
		"roman": func(n int) string {
			s, _ := num.Roman(n)
			return s
		},
		"alpha": func(n int) string {
			s, _ := num.Alpha(n)
			return strings.ToLower(s)
		},
		"add": func(nn ...int) (n int) {
			for _, m := range nn {
				n += m
			}
			return n
		},
		"hyphen": func(v interface{}) string {
			var s string
			switch v.(type) {
			case sd.NullString:
				s = v.(sd.NullString).String
			case string:
				s = v.(string)
			}
			return h.Hyphenate(s)
		},
		"mmYYYY": func(unixTimeStamp int64) string {
			if unixTimeStamp >= sd.PresentThreshold {
				return "Present"
			}
			layout := "Jan 2006"
			t := time.Unix(unixTimeStamp, 0)
			return t.Format(layout)
		},
		"date": func(unixTimeStamp int64) string {
			layout := "Jan 2, 2006"
			t := time.Unix(unixTimeStamp, 0)
			return t.Format(layout)
		},
		"squash": func(ii ...interface{}) []interface{} {
			return ii
		},
		"append": func(ii []interface{}, i ...interface{}) []interface{} {
			return append(ii, i...)
		},
		"skipThread": func(admin bool, p *sd.Post) bool {
			if admin {
				return false
			}
			showOp := !p.Deleted.Bool && p.PersVis == sd.VisibilityPublic
			if !showOp && len(p.Reply) == 0 {
				return true
			}
			return false
		},
		"postPrev": func(admin, op bool, p1, p2 *sd.Post) (show bool) {
			if op && admin {
				return true
			}
			if !op && p1.Created == p2.Created {
				return false
			}
			if p1.Deleted.Bool || p1.PersVis != sd.VisibilityPublic {
				return false
			}
			return true
		},
		"list": func(ss []string) string {
			return strings.Join(ss, ", ")
		},
		// As below but for a single value name.
		"mStr": func(s string, vv []sd.Value) string {
			for _, v := range vv {
				if s == v.Name {
					s = v.Text
					break
				}
			}
			return s
		},
		// Comma separted list of value names mapped to their display text.
		"mList": func(ss []string, vv []sd.Value) string {
			var mapped []string
			for _, v := range vv {
				if in(ss, v.Name) {
					mapped = append(mapped, v.Text)
				}
			}
			return strings.Join(mapped, ", ")
		},
		"lower": func(s string) string {
			return strings.ToLower(s)
		},
		"obfuscate": func(v interface{}) template.HTML {
			var s string
			switch v.(type) {
			case sd.NullString:
				s = v.(sd.NullString).String
			case string:
				s = v.(string)
			default:
				panic(`Unknown string type in template function "obfuscate"`)
			}
			var html string
			a := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
			for _, r := range s {
				tmpl := `<span class="obf">%s</span>`
				html += fmt.Sprintf(tmpl, string(r))
				html += fmt.Sprintf(tmpl, string(a[rand.Intn(len(a))]))
			}
			return template.HTML(html)
		},
		"rangeMeta": func(field sd.Field) interface{} {

			vv := field.Value

			var setting string
			if field.Text != "" {
				setting = field.Text
			} else {
				setting = field.Default
			}
			ss := strings.Split(setting, "-")

			var start int
			var end int
			for i, v := range vv {
				if v.Name == ss[0] {
					start = i
					end = i
				}
				if len(ss) > 1 && v.Name == ss[1] {
					end = i
				}
			}
			end++

			f := 100 / float64(len(vv)) * float64(start)
			ruleStart := template.CSS(fmt.Sprintf("left: %.2f%%; top: %.2f%%;", f, f))

			f = 100 / float64(len(vv)) * float64(end)

			/*
				If end is set to the last possible position we
				must substract the marker's width from offset.
			*/
			var ruleEnd template.CSS
			if end == len(vv) {
				ruleEnd = template.CSS(fmt.Sprintf(`
						left: calc(%.2f%% - var(--marker-brd));
						top:  calc(%.2f%% - var(--marker-brd));`,
					f, f))
			} else {
				ruleEnd = template.CSS(fmt.Sprintf("left: %.2f%%; top: %.2f%%;", f, f))
			}

			var status string
			if start != end-1 {
				status = fmt.Sprintf("%s – %s", vv[start].Text, vv[end-1].Text)
			} else {
				status = vv[start].Text
			}

			return struct {
				Status    string
				RuleStart template.CSS
				RuleEnd   template.CSS
				Start     int
				End       int
			}{
				Status:    status,
				RuleStart: ruleStart,
				RuleEnd:   ruleEnd,
				Start:     start,
				End:       end,
			}
		},
	})
	view.OnLoad(onSVGLoad)
	view.MustAddDir("", c.DirTemplates, nil, true)
	view.MustAddDir("", c.DirSVG, nil, true)
	return view
}
