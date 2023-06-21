package storydevs

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"html/template"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jakebowkett/go-gen/gen"
)

/*
TODO: make .Columns and .Values private and update
them exclusively through the SafeAdd method below.
*/
type DbTable struct {
	Name    string
	Slug    string
	Refs    int64
	Written []string
	Retain  []string
	Columns []string
	Values  []interface{}
	Tables  []*DbTable
	GoType  interface{}
}

/*
SafeAdd adds key-value pairs to the Columns and
Values slices of DbTable. It ensures that only
one instance of the pairing exists. If the caller
attempts to add a column that already exists the
value for that column will be updated - duplicate
entries will not be created.

It is strongly recommended to use this in
contexts where an action may be repeated such
as an operation being executed by a Tryer.
*/
func (dt *DbTable) SafeAdd(column string, value interface{}) {
	found := false
	for i, c := range dt.Columns {
		if c == column {
			dt.Values[i] = value
			found = true
		}
	}
	if !found {
		dt.Columns = append(dt.Columns, column)
		dt.Values = append(dt.Values, value)
	}
}

/*
AddSlug generates a pseudo-random slug
and adds it to DbTable using SafeAdd.
*/
func (dt *DbTable) AddSlug() string {
	slug, _ := gen.AlphaNum(11)
	dt.SafeAdd("slug", slug)
	return slug
}

type ResourceMapping map[string]map[string]string

type Resources map[string]ResourceService

/*
For retrieving stats about the overall
service or sub-elements thereof.
*/
type ResourceMeta interface {
	Meta(reqId, kind string, search Fields) error
}

type ResOpts struct {
	GetPrivate       bool
	CalledFromFilter bool
}

type ResourceService interface {
	Create(reqId string, r Resource, tt *DbTable) (Feedback, error)
	Retrieve(reqId, slug string, o ResOpts) (Resource, error)
	Filter(reqId string, admin bool, filter map[string][]string) (
		results []Resource,
		err error,
	)
	Update(reqId string, r Resource, tt *DbTable) (
		fb Feedback,
		toRemove []string,
		err error,
	)
	Delete(reqId, id string, persId int64) (
		fb Feedback,
		toRemove []string,
		err error,
	)
}

type Resource interface {
	IsOwner(account Account) bool

	GetAdmin() bool

	OwnerId() int64
	SetOwner(persona Persona)

	GetId() int64
	SetId(id int64)

	GetName() string
	GetHandle() string
	GetPersSlug() string

	GetSlug() string
	SetSlug(slug string)

	SetCreated(unixSeconds int64)
	SetUpdated(unixSeconds int64)

	GetVisibility() string

	SetHyphenator(h hyphenator)

	IsReply() bool
	IsLocked() bool
}

/*
	These fields contain server-side generated data and
	aren't subject to the resource validation process.
*/
type ResourceBase struct {
	Slug    string `ed:"ignore" validate:"ignore"`
	Created int64  `ed:"ignore" validate:"ignore"`
	Updated int64  `ed:"ignore" validate:"ignore"`
	Id      int64  `ed:"ignore" validate:"ignore" database:"ignore"`

	Hyphenate hyphenator `ed:"ignore" validate:"ignore" database:"ignore"`

	Admin        NullBool `ed:"ignore" validate:"ignore" database:"ignore"`
	AccId        int64    `ed:"ignore" validate:"ignore" database:"ignore"`
	PersId       int64    `ed:"ignore" validate:"ignore" database:"ignore"`
	PersVis      string   `ed:"ignore" validate:"ignore" database:"ignore"`
	PersSlug     string   `ed:"ignore" validate:"ignore" database:"ignore"`
	PersName     string   `ed:"ignore" validate:"ignore" database:"ignore"`
	PersHandle   string   `ed:"ignore" validate:"ignore" database:"ignore"`
	PersAvatar   FileName `ed:"ignore" validate:"ignore" database:"ignore"`
	PersPronouns []string `ed:"ignore" validate:"ignore" database:"ignore"`

	// Slug of a talent profile, if any exist for persona.
	PersProfile string `ed:"ignore" validate:"ignore" database:"ignore"`
}
type hyphenator = func(string) string

func (rb ResourceBase) GetAdmin() bool {
	return rb.Admin.Bool
}

func (rb ResourceBase) IsReply() bool {
	return false
}
func (rb ResourceBase) IsLocked() bool {
	return false
}

func (rb *ResourceBase) SetHyphenator(h hyphenator) {
	rb.Hyphenate = h
}

func (rb ResourceBase) IsOwner(account Account) bool {
	for _, persona := range account.Personas {
		if rb.PersId == persona.Id {
			return true
		}
	}
	return false
}

func (rb ResourceBase) GetHandle() string {
	return rb.PersHandle
}
func (rb ResourceBase) GetPersSlug() string {
	return rb.PersSlug
}
func (rb ResourceBase) OwnerId() (persId int64) {
	return rb.PersId
}
func (rb *ResourceBase) SetOwner(p Persona) {
	rb.Admin = p.Admin
	rb.AccId = p.AccId
	rb.PersId = p.Id
	rb.PersVis = p.Visibility
	rb.PersSlug = p.Slug
	rb.PersName = p.Name.String
	rb.PersHandle = p.Handle
	rb.PersAvatar = p.Avatar
	rb.PersPronouns = p.Pronouns
}
func (rb ResourceBase) GetId() int64 {
	return rb.Id
}
func (rb *ResourceBase) SetId(id int64) {
	rb.Id = id
}
func (rb ResourceBase) GetSlug() string {
	return rb.Slug
}
func (rb *ResourceBase) SetSlug(slug string) {
	rb.Slug = slug
}
func (rb *ResourceBase) SetCreated(unixSeconds int64) {
	rb.Created = unixSeconds
}
func (rb *ResourceBase) SetUpdated(unixSeconds int64) {
	rb.Updated = unixSeconds
}

var (
	VisibilityPublic   = "public"
	VisibilityUnlisted = "unlisted"
	VisibilityPrivate  = "private"
)

var TzSkip = []string{"utc", "local"}

type Event struct {
	ResourceBase

	Visibility string

	Name    NullString
	Summary NullString

	Body  RichText
	Words int `validate:"ignore" database:"ignore" ed:"ignore"`

	Timezone string
	Start    DateTime
	Finish   DateTime
	Weekly   NullBool

	Category []string
	Setting  []string
	Tag      []string
}

func (e Event) GenerateSummary() string {
	return generateSummary(e.Body, e.Hyphenate)
}

func (e Event) BodyHTML() template.HTML {
	return richTextToHTML(e.Body, e.Hyphenate, false)
}

func (e Event) Deltas() (*EventDeltas, error) {

	now := time.Now()
	evtStart := time.Unix(int64(e.Start.DateTime), 0)
	evtFinish := time.Unix(int64(e.Finish.DateTime), 0)
	lasting := time.Second * time.Duration(e.Finish.DateTime-e.Start.DateTime)
	until := evtStart.Sub(now)

	/*
		This can be negative if we're within the period the
		event occurs so we make sure to clamp it to zero.
	*/
	if until < 0 {
		until = 0
	}

	us := evtStart.Unix()
	uf := evtFinish.Unix()
	h := int64(until.Hours())
	d := h / 24
	w := d / 7
	m := int64(until.Minutes()) - h*60
	s := int64(until.Seconds()) - h*60*60 - m*60

	if e.Finish.Null {
		lasting = 0
	}

	return &EventDeltas{
		Start: us,
		Finish: NullInt64{
			Int64: uf,
			Null:  e.Finish.Null,
		},
		Months:  d / 30,
		Weeks:   w,
		Days:    d,
		Hours:   h % 24,
		Minutes: m,
		Seconds: s,
		lasting: lasting,
	}, nil
}

type EventDeltas struct {
	Start   int64
	Finish  NullInt64
	Months  int64
	Weeks   int64
	Days    int64
	Hours   int64
	Minutes int64
	Seconds int64
	lasting time.Duration
}

// First slice value contains label, second is value.
func (ed EventDeltas) Until() EventUntil {
	now := time.Now()
	if ed.Finish.Null {
		if s := time.Unix(ed.Start, 0); now.After(s) {
			return EventUntil{Label: "Has Occurred"}
		}
	} else {
		if f := time.Unix(ed.Finish.Int64, 0); now.After(f) {
			return EventUntil{Label: "Is over"}
		}
		if s := time.Unix(ed.Start, 0); now.After(s) {
			return EventUntil{Label: "Has begun"}
		}
	}
	u := ""
	switch {
	case ed.Months > 1:
		u = fmt.Sprintf("%d months", ed.Months)
	case ed.Months == 1:
		u = "1 month"
	case ed.Weeks > 1:
		u = fmt.Sprintf("%d weeks", ed.Weeks)
	case ed.Weeks == 1:
		u = "1 week"
	case ed.Days > 1:
		u = fmt.Sprintf("%d days", ed.Days)
	case ed.Days == 1:
		u = "1 day"
	case ed.Hours > 1:
		u = fmt.Sprintf("%d hours", ed.Hours)
	case ed.Hours == 1:
		u = "1 hour"
	case ed.Minutes > 1:
		u = fmt.Sprintf("%d mins", ed.Minutes)
	case ed.Minutes == 1:
		u = "1 min"
	case ed.Seconds > 1:
		u = fmt.Sprintf("%d secs", ed.Seconds)
	case ed.Seconds == 1:
		u = "1 sec"
	}
	return EventUntil{
		Label: "In ",
		Unit:  u,
	}
}

type EventUntil struct {
	Label string
	Unit  string
}

func (ed EventDeltas) Lasting(compact bool) string {

	h := int(ed.lasting.Hours())
	d := h / 24
	h %= 24
	m := int(ed.lasting.Minutes()) % 60

	args := []struct {
		val     int
		unit    string
		compact string
	}{
		{val: d, unit: "day", compact: "day"},
		{val: h, unit: "hour", compact: "hr"},
		{val: m, unit: "minute", compact: "min"},
	}

	var ss []string
	for _, a := range args {
		if len(ss) == 2 {
			break
		}
		if a.val == 0 {
			continue
		}
		unit := a.unit
		if compact {
			unit = a.compact
		}
		plural := ""
		if a.val > 1 {
			plural = "s"
		}
		ss = append(ss, strconv.Itoa(a.val)+" "+unit+plural)
	}

	comma := ","
	if compact {
		comma = ""
	}
	return strings.Join(ss, comma+" ")
}

func (e Event) GetVisibility() string {
	return e.Visibility
}
func (e *Event) SetVisibility(visibility string) {
	e.Visibility = visibility
}
func (e Event) GetName() string {
	return e.Name.String
}

type Profile struct {
	ResourceBase

	Available  bool
	Visibility string

	Name    NullString
	Summary NullString

	Website NullString
	Email   NullString
	Discord NullString

	Duration Range

	Tag          []string
	Compensation []string
	Medium       []string
	Language     []string
	Project      []Project
	Advertised   []Advertised
}

func (p Profile) GetVisibility() string {
	return p.Visibility
}
func (p *Profile) SetVisibility(visibility string) {
	p.Visibility = visibility
}

type Range struct {
	Start string
	End   string
}

func (r *Range) UnmarshalJSON(p []byte) error {

	/*
		Raw JSON strings include double quotes around
		them so we strip those here.
	*/
	s := string(p)[1 : len(p)-1]

	ss := strings.Split(s, "-")
	r.Start = ss[0]
	if len(ss) == 2 {
		r.End = ss[1]
	} else {
		r.End = ss[0]
	}
	return nil
}

func (p Profile) GetName() string {
	return p.Name.String
}

func (p Profile) ProjectNames() (names []string) {
	for _, pjt := range p.Project {
		names = append(names, pjt.Name)
	}
	return names
}

type Project struct {
	Name     string
	Link     NullString
	TeamName NullString
	TeamLink NullString
	Start    int64
	Finish   int64
	Role     []Role
}
type Role struct {
	Name    string
	Comment NullString
	Skill   []string
	Duty    []string
}
type Advertised struct {
	Skill   string
	Example []Media
}
type Media struct {
	AltText string // for images
	Title   string
	Project string
	Info    string

	// persona handle - populated by validator, added as metadata to .File
	Artist string `ed:"ignore" validate:"ignore" database:"ignore"`

	// text, code, image, audio, video - populated by validator
	Kind string `ed:"ignore" validate:"ignore"`

	// jpeg, png, mp3, etc - populated by validator
	Format string `ed:"ignore" validate:"ignore"`

	// equals width / height - portrait is < 1, landscape is > 1
	Aspect float64 `ed:"ignore" validate:"ignore"`

	// image, audio, video
	File File `ed_ref:"Kind" ed_wrap:"example" ed_rm:"example" validate:"ignore" database:"ignore"`

	// rich editors - text, code; tables manually created (i.e, still added to DB)
	RichText []Paragraph `ed:"ignore" validate:"ignore" database:"ignore"`
}

type ReadSeekCloser interface {
	Read(p []byte) (n int, err error)
	Seek(offset int64, whence int) (int64, error)
	Close() error
}

/*
Aspects are width:height ratio. These are meant to
describe the best fitting container aspect for an
image in a grid.
*/
const (
	AspectSquare    = "1:1"
	AspectLandscape = "2:1"
	AspectThicc     = "3:1"
	AspectPortrait  = "1:2"
	AspectTallBoy   = "1:3"
)

const (
	MediaText  = "text"
	MediaCode  = "code"
	MediaImage = "image"
	MediaAudio = "audio"
	MediaVideo = "video"
)

const (
	FormatJPEG = "jpeg"
	FormatPNG  = "png"
)

var FileFormats = []string{
	FormatJPEG,
	FormatPNG,
}

type Threader interface {
	GetThread() string

	/*
	   Items returns a slice of Resource that represents
	   all the Resource that comprise a thread, including
	   the root Resource.
	*/
	Items() []Resource
}

func (p *Post) Items() []Resource {
	rr := make([]Resource, 0, len(p.Reply)+1)
	rr = append(rr, p)
	for i := range p.Reply {
		rr = append(rr, &p.Reply[i])
	}
	return rr
}

type Post struct {
	ResourceBase

	Deleted NullBool `validate:"ignore" database:"ignore" ed:"ignore"`

	Pinned NullBool
	Locked NullBool

	Visibility string

	Name    NullString `reply:"ignore"`
	Summary NullString `reply:"ignore"`

	Body  RichText
	Words int `validate:"ignore" database:"ignore" ed:"ignore"`

	/*
		What modes this post belongs to i.e.
		"library", "forums", etc.
	*/
	Kind []string `validate:"ignore" database:"ignore" ed:"ignore"`

	Category []string `reply:"ignore"`

	Tag []string `reply:"ignore"`

	/*
		ThreadSlug is only required for replies.
	*/
	ThreadSlug string `validate:"ignore" database:"ignore" ed:"ignore"`

	/*
		Used to compare with post's id to determine
		if it's the root of the thread.
	*/
	ThreadId int64 `validate:"ignore" database:"ignore" ed:"ignore"`

	/*
		List of other posts this one references.
	*/
	// Ref []int64 ``

	/*
		Reply is only used for retrieval,
		not creating/updating/deleting.
	*/
	Reply []Post `validate:"ignore" database:"ignore" ed:"ignore"`
}

/*
For non-admin users this is the number of public
replies to p - it excludes deleted and posts by
non-public personas. For admins it is the total
number of replies ever made to this thread.
*/
func (p Post) ReplyCount(admin bool) int {
	if admin {
		return len(p.Reply)
	}
	n := 0
	for _, r := range p.Reply {
		isPublic := r.PersVis == VisibilityPublic
		deleted := r.Deleted.Bool
		if !deleted && isPublic {
			n++
		}
	}
	return n
}

/*
LastReply is used in browse.html to get the
date of the last post to a thread.
*/
func (p Post) LastReply() *Post {
	if len(p.Reply) == 0 {
		return &p
	}
	i := len(p.Reply) - 1
	for ; i >= 0; i-- {
		isPublic := p.Reply[i].PersVis == VisibilityPublic
		deleted := p.Reply[i].Deleted.Bool
		if !deleted && isPublic {
			break
		}
	}
	if i < 0 {
		return &p
	}
	return &p.Reply[i]
}

func generateSummary(rt RichText, h hyphenator) string {
	summary := ""
	n := 0
	max := 200
	truncated := false
outer:
	for _, para := range rt {
		for _, span := range para.Span {
			diff := max - n
			n += len(span.Text)
			if n >= max {
				// Clamp end so it doesn't go out of bounds.
				end := diff
				if end > len(span.Text) {
					end = len(span.Text)
				}
				summary += span.Text[0:end]
				truncated = true
				break outer
			}
			summary += span.Text
		}
		summary += " "
	}
	summary = summary[:len(summary)-1] // Remove last space.
	if truncated {
		summary += "..."
	}
	if h == nil {
		return summary
	}
	return h(summary)
}

func (p Post) GenerateSummary() string {
	return generateSummary(p.Body, p.Hyphenate)
}

func (p Post) IsReply() bool {
	return p.Id != p.ThreadId
}
func (p Post) IsLocked() bool {
	return p.Locked.Bool
}

func (p Post) BodyHTML() template.HTML {
	return richTextToHTML(p.Body, p.Hyphenate, false)
}

func (p Post) GetThread() string {
	return p.ThreadSlug
}

func (p Post) GetVisibility() string {
	return p.Visibility
}
func (p *Post) SetVisibility(visibility string) {
	p.Visibility = visibility
}
func (p Post) GetName() string {
	return p.Name.String
}

func (p Post) ReadingTime(wordsPerMin int) string {

	wpm := float64(wordsPerMin)
	rt := float64(p.Words) / wpm

	if rt < 1 {
		rt = 60 * rt
		return fmt.Sprintf("%ds", int(rt))
	}

	if rt >= 60 {
		rt = rt / 60
		unit := "hr"
		return fmt.Sprintf("%.1f %s", rt, unit)
	}

	unit := "min"
	return fmt.Sprintf("%d %s", int(rt), unit)
}

type Paragraph struct {
	Kind string `validate:"ignore"`
	Span []Span `validate:"ignore"`
}
type Span struct {
	Format []string
	Link   NullString
	Text   string
}

type RichText []Paragraph

func (rt RichText) HTML() template.HTML {
	return richTextToHTML(rt, nil, true)
}

func makeAnchor(s string) string {

	var ss []string
	skip := []string{
		"a",
		"an",
		"the",
	}
	for _, word := range strings.Fields(s) {
		if in(skip, word) {
			continue
		}
		ss = append(ss, word)
	}
	s = strings.Join(ss, "-")

	letterOrNumber := regexp.MustCompile(`[^\pL|\d|-]+`)
	s = letterOrNumber.ReplaceAllString(s, "")
	s = strings.ToLower(s)
	s = url.PathEscape(s)

	rr := []rune(s)
	if len(rr) > 24 {
		rr = rr[:24]
	}
	s = string(rr)
	s = strings.Trim(s, "-")

	md5Bytes := md5.Sum([]byte(s))
	s += "-"
	s += hex.EncodeToString(md5Bytes[:4])

	return s
}

/*
Should probably be doing this in an actual template. We're
dealing with user submitted content and it's very easy to
carelessly convert a malicious string into html, allowing
an injection attack.

Both links and text are escaped while formats and paragraph
kinds are validated against a whitelist. So it should be safe
at the moment but it may be tempting to extend this in the
future and carelessly introduce a serious bug.
*/
func richTextToHTML(rt RichText, h hyphenator, inEditor bool) template.HTML {

	html := ""

	for i, p := range rt {

		inList := in([]string{"ul", "ol"}, p.Kind)
		prevKind, nextKind := neighbourParas(rt, i)
		kind := p.Kind

		if inList {
			kind = "li"
			if p.Kind != prevKind {
				html += "<" + p.Kind + ">"
			}
		}

		/*
			While makeAnchor isn't cleaned with HTMLEscapeString,
			it does have everything except letters, numbers, and
			hyphens removed. Then url.PathEscape is called on it.
		*/
		anchor := ""
		if !inEditor && kind == "h2" {
			anchor = fmt.Sprintf(` id="%s"`, makeAnchor(p.Span[0].Text))
		}

		html += fmt.Sprintf("<%s%s>", kind, anchor)

		for _, s := range p.Span {
			tag := "span"
			link := template.HTMLEscapeString(s.Link.String)
			if s.Link.String != "" {
				if inEditor {
					s.Format = append(s.Format, "a")
					link = ` data-link="` + link + `"`
				} else {
					tag = "a"
					link = ` href="` + link + `" target="_blank" rel="noreferrer noopener"`
				}
			}
			f := strings.Join(s.Format, " ")
			if s.Format != nil {
				f = ` class="` + f + `"`
			}
			html += fmt.Sprintf("<%s%s%s>", tag, link, f)
			text := s.Text
			if !inEditor && h != nil {
				/*
					For now we don't hyphenate text. Formatting can
					split words up in such a way that they will be
					inconsistently hyphenated, if at all.

					We can probably do something like, say, concatenate
					all the spans in a paragraph then hyphenate /that/
					and index into it somehow??? I don't know.

					Anyway, that's why we're not hyphenating here yet.
					Would like to though.
				*/
				// text = h(text)
			}
			html += template.HTMLEscapeString(text)
			html += fmt.Sprintf("</%s>", tag)
		}

		html += "</" + kind + ">"

		if inList && p.Kind != nextKind {
			html += "</" + p.Kind + ">"
		}
	}
	return template.HTML(html)
}

func neighbourParas(rt RichText, i int) (string, string) {
	var prevKind string
	var nextKind string
	if i != 0 {
		prevKind = rt[i-1].Kind
	}
	if i != len(rt)-1 {
		nextKind = rt[i+1].Kind
	}
	return prevKind, nextKind
}

func (rt RichText) Words() int {
	s := ""
	for i, p := range rt {
		for _, span := range p.Span {
			s += span.Text
		}
		if i != len(rt)-1 {
			s += "\n"
		}
	}
	whitespace := regexp.MustCompile(`\s+`)
	return len(whitespace.Split(s, -1))
}
