package storydevs

import (
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type DbRow = *sqlx.Row
type DbRows = *sqlx.Rows
type DbResult = sql.Result

type DB interface {
	Get(dest interface{}, q string, args ...interface{}) error
	Select(dest interface{}, q string, args ...interface{}) error
	Query(q string, args ...interface{}) (rows DbRows, err error)
	Exists(fromWhere string, args ...interface{}) (exists bool, err error)
	Exec(q string, args ...interface{}) (result DbResult, err error)
	Begin() (Tx, error)
	BeginRead() (Tx, error)
	Close() error
}

type Tx interface {
	Get(dest interface{}, q string, args ...interface{}) error
	Select(dest interface{}, q string, args ...interface{}) error
	Query(q string, args ...interface{}) (rows DbRows, err error)
	Exists(fromWhere string, args ...interface{}) (exists bool, err error)
	Exec(q string, args ...interface{}) (result DbResult, err error)
	Rollback(err error) error
	Commit() error
}

func ReflectKind(rv reflect.Value) reflect.Kind {
	kind := rv.Kind()
	typ := rv.Type().Name()
	switch typ {
	case "NullString":
		kind = reflect.String
	case "NullBool":
		kind = reflect.Bool
	case "NullInt64":
		kind = reflect.Int64
	case "NullFloat64":
		kind = reflect.Float64
	}
	return kind
}

type FileName string

func (fn FileName) String() string {
	return string(fn)
}

func (fn FileName) URLThumb() string {
	if fn == "" {
		return ""
	}
	ext := filepath.Ext(string(fn))
	name := strings.TrimSuffix(string(fn), ext)
	return "/user/" + name + "_thumb" + ext
}

func (fn FileName) URL() string {
	if fn == "" {
		return ""
	}
	return "/user/" + string(fn)
}

func (fn FileName) URLFull() string {
	return "https://storydevs.com" + string(fn.URL())
}

func (fn *FileName) Set(fileName string) {
	*fn = FileName(fileName)
}

func (fn *FileName) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		*fn = ""
	case string:
		*fn = FileName(src)
	case []uint8:
		*fn = FileName(src)
	default:
		return errors.New("src is not of type string, []uint8, or nil")
	}
	return nil
}

type File struct {
	Data ReadSeekCloser
	Name FileName
}

func (f *File) SetFile(rsc ReadSeekCloser) {
	f.Data = rsc
}

func (f *File) UnmarshalJSON(src []byte) error {
	s := string(src)
	f.Data = nil
	if s == "null" {
		f.Name = ""
		return nil
	}
	s = s[1 : len(s)-1] // Strings are quoted and must be stripped.
	f.Name = FileName(s)
	return nil
}

func (f *File) Scan(src interface{}) error {
	f.Data = nil
	switch src := src.(type) {
	case nil:
		f.Name = ""
	case string:
		f.Name = FileName(src)
	case []uint8:
		f.Name = FileName(src)
	default:
		return errors.New("src is not of type string, []uint8, or nil")
	}
	return nil
}

// This is also called in /setup/templates.go
func TimeValue(n int64) []string {
	d := time.Duration(n) * time.Second
	h := int(d.Hours())
	m := int(d.Minutes()) - (h * 60)
	meridiem := "am"
	if h >= 12 {
		meridiem = "pm"
	}
	if h > 12 {
		h -= 12
	}
	if h == 0 {
		h = 12
	}
	s := fmt.Sprintf("%02d%02d%s", h, m, meridiem)
	return []string{s[:2], s[2:4], s[4:6]}
}

type DateTime struct {
	DateTime int64
	TZOff    int
	Null     bool
}

func UTC(sec int64) time.Time {
	t := time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	return t.Add(time.Second * time.Duration(sec))
}

func (dt DateTime) Date() string {
	t := UTC(dt.DateTime)
	return t.Format("January 2, 2006")
}
func (dt DateTime) Month() string {
	t := UTC(dt.DateTime)
	return t.Format("January 2006")
}
func (dt DateTime) Time() []string {
	days := ((dt.DateTime / 60) / 60) / 24
	return TimeValue(dt.DateTime - days)
}
func (dt *DateTime) UnmarshalJSON(src []byte) error {
	s := string(src)
	if s == "null" {
		dt.Null = true
		dt.DateTime = 0
	}
	ss := extractNum.FindAllString(s, -1)
	if len(ss) != 2 {
		return fmt.Errorf("invalid amount of numbers found in sd.DateTime JSON (%d)", len(ss))
	}
	n1, err := strconv.ParseInt(ss[0], 10, 64)
	if err != nil {
		return err
	}
	n2, err := strconv.ParseInt(ss[1], 10, 64)
	if err != nil {
		return err
	}
	dt.DateTime = n1 + n2
	return nil
}
func (dt *DateTime) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		dt.Null = true
		dt.DateTime = 0
	case int64:
		dt.Null = false
		dt.DateTime = src
	default:
		return errors.New("src is not of type int64 or nil")
	}
	return nil
}

var extractNum = regexp.MustCompile(`(\d+)`)

type NullString struct {
	String string
	Null   bool
}

func (ns *NullString) UnmarshalJSON(src []byte) error {
	s := string(src)
	if s == "null" {
		ns.Null = true
		ns.String = ""
		return nil
	}
	if len(s) < 2 || s[0:1] != `"` || s[len(s)-1:] != `"` {
		return errors.New("src is not of type string")
	}
	s = s[1 : len(s)-1] // Strings are quoted and must be stripped.
	ns.String = s
	ns.Null = false
	return nil
}

func (ns *NullString) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		ns.Null = true
		ns.String = ""
	case string:
		ns.String = src
		ns.Null = false
	case []uint8:
		ns.String = string(src)
		ns.Null = false
	default:
		return errors.New("src is not of type string, []uint8, or nil")
	}
	return nil
}

type NullBool struct {
	Bool bool
	Null bool
}

func (nb *NullBool) UnmarshalJSON(src []byte) error {
	s := string(src)
	switch s {
	case "null":
		nb.Null = true
		nb.Bool = false
	case "true":
		nb.Null = false
		nb.Bool = true
	case "false":
		nb.Null = false
		nb.Bool = false
	default:
		return errors.New("src is not of type bool")
	}
	return nil
}

func (nb *NullBool) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		nb.Null = true
		nb.Bool = false
	case bool:
		nb.Bool = src
		nb.Null = false
	default:
		return errors.New("src is not of type bool or nil")
	}
	return nil
}

type NullInt64 struct {
	Int64 int64
	Null  bool
}

func (ni *NullInt64) UnmarshalJSON(src []byte) error {
	s := string(src)
	if s == "null" {
		ni.Null = true
		ni.Int64 = 0
		return nil
	}
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("src cannot be unmarshaled to type int64 or nil: %w", err)
	}
	ni.Int64 = i
	ni.Null = false
	return nil
}

func (ni *NullInt64) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		ni.Null = true
		ni.Int64 = 0
	case int64:
		ni.Null = false
		ni.Int64 = src
	default:
		return errors.New("src is not of type int64 or nil")
	}
	return nil
}

type NullFloat64 struct {
	Float64 float64
	Null    bool
}

func (nf *NullFloat64) UnmarshalJSON(src []byte) error {
	s := string(src)
	if s == "null" {
		nf.Null = true
		nf.Float64 = 0
		return nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("src cannot be unmarshaled to type float64 or nil: %w", err)
	}
	nf.Float64 = f
	nf.Null = false
	return nil
}

func (nf *NullFloat64) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		nf.Null = true
		nf.Float64 = 0
	case float64:
		nf.Float64 = src
		nf.Null = false
	default:
		return errors.New("src is not of type float64 or nil")
	}
	return nil
}
