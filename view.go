package storydevs

import (
	"bytes"
	"fmt"
	"html/template"
	"regexp"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

type Hyphenator interface {
	Hyphenate(text string) string
}

type View interface {
	MustAddDir(alias, dirPath string, ext []string, recursive bool)
	Render(alias string, data interface{}) (tmpl []byte, err error)
	Refresh() (dropped []string)
	List() []string
}

type Base struct {
	View     string
	SubView  string
	ViewType string

	Title       string // used for tab title and meta title
	MetaDesc    string // description for social media embeds
	MetaCard    string // url pointing to image for social media embeds
	MetaAlt     string // alt text for above image
	MetaURL     string
	MetaTwitter string // handle of person URL references

	Layout        string
	Columns       string
	Editing       string
	ResourceSlug  string
	ResourceOwner bool
	ViewMeta      *ViewData
	JavaScript    template.JS
	Styling       template.CSS
	Page          template.HTML
	Modal         template.HTML
	Search        Column
	Browse        Column
	Detail        Column
	Editor        Column
	Account       Account

	PresentThreshold int64
}
type BaseMeta struct {
	Title    string
	Resource string
}
type Column struct {
	Name    string
	Empty   string
	Content template.HTML
}

type ViewData struct {
	Page       Page
	Mode       Mode
	Modal      Modal
	Errors     map[string]string
	Hyphenator hyphenator
}

type Page = map[string]PageData
type PageData struct {
	Name     string
	Title    string
	Message  string
	Disabled bool
	Data     interface{}
}

type EditorKind string

var EdNew EditorKind = "new"
var EdEdit EditorKind = "edit"
var EdReply EditorKind = "reply"

type Mode = map[string]ModeData
type ModeData struct {
	Account        Account
	AdminOnly      bool // Is the mode only accessible to admins?
	IsAdmin        bool // Is the active persona an admin?
	InAdmin        bool // Are we in the admin mode?
	InAccount      bool // Are we in the account mode?
	LogoutRemove   bool
	Disabled       bool
	Name           string
	Title          string
	BrowseName     string
	ResourceName   string
	ResourcePlural string
	ResourceColumn string
	EditorKind     EditorKind
	Search         Fields
	Editor         Fields
	Results        []Resource
	Resource       Resource
}

type Replace struct {
	Dir  string
	Load func(dir string, data interface{}) error
}

func (r Replace) InstanceOf(key string) (f Field, err error) {

	m := make(map[string]Field)
	err = r.Load(r.Dir, &m)
	if err != nil {
		return f, err
	}

	f, ok := m[key]
	if !ok {
		return f, fmt.Errorf("no such field %q in Replace", key)
	}

	return f, nil
}

type Shared = map[string][]Value

type Fields []Field

func (mg Fields) FieldById(id string) (*Field, error) {
	if f := fieldById(id, []Field(mg)); f != nil {
		return f, nil
	}
	return nil, fmt.Errorf("no field with Id %q", id)
}
func fieldById(id string, ff []Field) *Field {
	for _, f := range ff {
		if f.Id == id {
			return &f
		}
		if len(f.Field) > 0 {
			if f := fieldById(id, f.Field); f != nil {
				return f
			}
		}
	}
	return nil
}

func in(ss []string, s string) bool {
	for i := range ss {
		if ss[i] == s {
			return true
		}
	}
	return false
}

func (mg Fields) SetInit(name string) (f *Field, err error) {
	f, err = mg.Field(name)
	if err != nil {
		return nil, err
	}
	f.Disabled = false
	f.RequestOnly = false
	return f, nil
}

func (mg Fields) WrapField(name, wrap string) error {
	f, err := mg.SetInit(name)
	if err != nil {
		return err
	}
	f.Wrap = wrap
	return nil
}

func (mg Fields) RemoveField(name string) error {
	parts := strings.Split(strings.ToLower(name), ".")
	f, ok := findField([]Field(mg), parts[0:len(parts)-1])
	if !ok {
		return fmt.Errorf(`no such field %q`, name)
	}
	if ok := removeField(&f.Field, parts[len(parts)-1]); !ok {
		return fmt.Errorf(`no such field %q`, name)
	}
	return nil
}

func removeField(ff *[]Field, name string) bool {
	for i, f := range *ff {
		if f.Name == name {
			*ff = append((*ff)[0:i], (*ff)[i+1:]...)
			return true
		}
	}
	return false
}

// AddFieldInstance finds the last field that matches
// name and adds another field of the same name
// after it.
func (mg Fields) AddFieldInstance(name string) error {
	f, err := mg.SetInit(name)
	if err != nil {
		return err
	}
	return AddFieldInstance(f)
}

func AddFieldInstance(f *Field) error {

	// Copy field.
	var instance Field
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(f); err != nil {
		return err
	}
	if err := toml.Unmarshal(buf.Bytes(), &instance); err != nil {
		return err
	}

	// Set instances to nil.
	instance.Instances = nil

	// Append to instances.
	f.Instances = append(f.Instances, instance)

	return nil
}

func (mg Fields) SetWithBool(name string, b bool) error {

	f, err := mg.SetInit(name)
	if err != nil {
		return err
	}

	vv := f.Value
	valueSet := false
	defaultAt := 0
	for i := range vv {
		if vv[i].Default {
			defaultAt = i
		}
		vv[i].Default = false
		if vv[i].True && b {
			vv[i].Default = true
			valueSet = true
		}
		if !vv[i].True && !b {
			vv[i].Default = true
			valueSet = true
		}
	}

	if !valueSet {
		vv[defaultAt].Default = true
	}

	return nil
}

func (mg Fields) SetWithInt(name string, n int64) error {

	f, err := mg.SetInit(name)
	if err != nil {
		return err
	}

	switch f.Type {
	case "calendar":
		if n >= PresentThreshold {
			f.Text = "Present"
		} else {
			f.Text = UTC(n).Format("January 2006")
		}
	case "date":
		if n >= PresentThreshold {
			f.Text = "Present"
		} else {
			f.Text = UTC(n).Format("January 2, 2006")
		}
	case "time":
		f.Text = strconv.FormatInt(n, 10)
	}

	return nil
}

func (mg Fields) SetWithString(name, s string) error {

	f, err := mg.SetInit(name)
	if err != nil {
		return err
	}

	if f.Type == "radio" {

		vv := f.Value
		valueSet := false
		defaultAt := 0
		for i := range vv {
			if vv[i].Default {
				defaultAt = i
			}
			vv[i].Default = false
			if vv[i].Name == s {
				vv[i].Default = true
				valueSet = true
				break
			}
		}

		if !valueSet {
			vv[defaultAt].Default = true
		}

		return nil
	}

	if f.Type == "dropdown" {

		vv := f.Value

		// If the dropdown draws its values from another field.
		if f.Ref != "" {
			ref, err := mg.Field(f.Ref)
			if err != nil {
				return err
			}
			vv = ref.Value
		}

		for _, v := range vv {
			if v.Name == s {
				f.Icon = v.Icon
				if len(v.Value) == 0 {
					s = v.Text
				} else {
					var ss []string
					for i := range v.Value {
						ss = append(ss, v.Value[i].Text)
					}
					ss = append(ss, v.Text)
					s = strings.Join(ss, " ")
				}
				break
			}
		}
	}

	f.Text = s

	return nil
}

func (mg Fields) SetWithSlice(name string, ss []string) error {

	f, err := mg.SetInit(name)
	if err != nil {
		return err
	}

	if f.Type == "checkbox" {
		vv := f.Value
		for i := range vv {
			if in(ss, vv[i].Name) {
				vv[i].Default = true
			}
		}
		return nil
	}

	if f.Type == "tagger" {
		for _, s := range ss {
			f.Value = append(f.Value, Value{Text: s})
		}
		return nil
	}

	for i, s := range ss {
		if err := mg.AddFieldInstance(name); err != nil {
			return err
		}
		idx := fmt.Sprintf("%s.%d", name, i)
		if err := mg.SetWithString(idx, s); err != nil {
			return err
		}
	}

	return nil
}

var endsWithIndex = regexp.MustCompile(`\.\d+$`)

func (mg Fields) FindOrAddInstance(path string) (*Field, error) {
	if endsWithIndex.MatchString(path) {
		return nil, fmt.Errorf("path %q ends with index", path)
	}
	parts := strings.Split(path, ".")
	for _, g := range []Field(mg) {
		f, err := mg.findOrAddInstance(g.Field, parts, 0)
		if err != nil {
			continue
		}
		return f, nil
	}
	return nil, fmt.Errorf("FindOrAddInstance: could not find %q", path)
}

func (mg Fields) findOrAddInstance(ff []Field, parts []string, n int) (*Field, error) {

	for i := range ff {
		if !fieldMatch(ff, parts[n], i) {
			/*
				Treat fields belonging to subgroups that
				cannot have multiple instances as being
				siblings to the enclosing groups' fields.
			*/
			if ff[i].Add == 0 {
				f, err := mg.findOrAddInstance(ff[i].Field, parts, n)
				if err == nil {
					return f, nil
				}
			}
			continue
		}
		if n == len(parts)-1 {
			return &ff[i], nil
		}

		// Look ahead at next segment to see if it's an instance.
		n++
		var subFields []Field
		if isNum.MatchString(parts[n]) {
			count, err := strconv.Atoi(parts[n])
			if err != nil {
				return nil, err
			}
			if ff[i].Add > 0 && count >= len(ff[i].Instances) {
				err := mg.AddFieldInstance(strings.Join(parts[:n], "."))
				if err != nil {
					return nil, err
				}
			}
			subFields = ff[i].Instances
		} else {
			subFields = ff[i].Field
		}

		f, err := mg.findOrAddInstance(subFields, parts, n)
		if err == nil {
			return f, nil
		}
	}

	return nil, fmt.Errorf(
		"findOrAddInstance: could not find %q",
		strings.Join(parts, "."),
	)
}

func (mg Fields) Field(name string) (f *Field, err error) {
	parts := strings.Split(strings.ToLower(name), ".")
	f, ok := findField([]Field(mg), parts)
	if !ok {
		return nil, fmt.Errorf(`no such field %q`, name)
	}
	return f, nil
}

var isNum = regexp.MustCompile(`^\d+$`)

func fieldMatch(ff []Field, part string, i int) bool {
	if part == "*" {
		return true
	}
	if ff[i].Name == part {
		return true
	}
	if i == partNum(part) {
		return true
	}
	return false
}

func partNum(part string) int {
	n, err := strconv.Atoi(part)
	if err != nil {
		return -1
	}
	return n
}

func findField(ff []Field, parts []string) (f *Field, ok bool) {

	if len(parts) == 0 {
		return nil, false
	}

	for i := range ff {

		if !fieldMatch(ff, parts[0], i) {
			/*
				Treat fields belonging to subgroups that
				cannot have multiple instances as being
				siblings to the enclosing groups' fields.
			*/
			if ff[i].Add == 0 {
				f, ok := findField(ff[i].Field, parts)
				if ok {
					return f, true
				}
			}
			continue
		}

		if len(parts) == 1 {
			return &ff[i], true
		}

		// Look ahead to next part and check if it's an index.
		parts := parts[1:]
		if len(parts) == 0 {
			break
		}
		var subFields []Field
		if isNum.MatchString(parts[0]) {
			subFields = ff[i].Instances
		} else {
			subFields = ff[i].Field
		}

		f, ok := findField(subFields, parts)
		if ok {
			return f, true
		}
	}

	return nil, false
}

func (mg Fields) Value(fieldName, valueName string) (*Value, error) {
	// func (mg Fields) Value(fieldName, valueText string) (*Value, error) {
	f, err := mg.Field(fieldName)
	if err != nil {
		return nil, err
	}
	return f.ValueByName(valueName)
	// return f.FindValue(valueText)
}
func (mg Fields) ValuesData(name string) ([]string, error) {
	f, err := mg.Field(name)
	if err != nil {
		return nil, err
	}
	dd := f.ValueData()
	return dd, nil
}
func (mg Fields) ValuesText(name string) ([]string, error) {
	f, err := mg.Field(name)
	if err != nil {
		return nil, err
	}
	return f.ValueText(), nil
}
func (f Field) Find(name string) (*Field, error) {
	parts := strings.Split(strings.ToLower(name), ".")
	result, ok := findField(f.Field, parts)
	if !ok {
		return nil, fmt.Errorf(`no such field %q`, name)
	}
	return result, nil
}
func (f Field) FindValue(valueText string) (*Value, error) {
	for _, v := range f.Value {
		if v.Text == valueText {
			return &v, nil
		}
	}
	return nil, fmt.Errorf("unable to find value with text %q", valueText)
}
func (f Field) ValueByName(name string) (*Value, error) {
	for _, v := range f.Value {
		if v.Name == name {
			return &v, nil
		}
	}
	return nil, fmt.Errorf("unable to find value with name %q", name)
}
func (f Field) ValueData() (dd []string) {
	for _, v := range f.Value {
		dd = append(dd, v.Data)
	}
	return dd
}
func (f Field) ValueText() (vv []string) {
	for _, v := range f.Value {
		vv = append(vv, v.Text)
	}
	return vv
}
func (f Field) ValueNames() (vv []string) {
	for _, v := range f.Value {
		vv = append(vv, v.Name)
	}
	return vv
}
func (f Field) InValue(text string) bool {
	for _, v := range f.Value {
		if v.Name == text {
			return true
		}
	}
	return false
}

/*
	Form data structures.
*/
type Modal = map[string]ModalData
type ModalData struct {
	Name     string
	Title    string
	Class    string
	Msg      string
	Field    Fields
	Button   []Button
	Disabled bool
	IsAdmin  bool
}

type FieldUpdateFunc func(*Field) error
type FieldUpdaters map[string]FieldUpdateFunc

type Field struct {
	Icon        template.HTML // header icon for groups, icon for dropdown fields
	AddIcon     template.HTML // graphic for adding new group instances
	AddName     string        // text for adding multiple instances, e.g. "+ Add [name]"
	Id          string        // used for compact queries
	Name        string        // used for HTML element name attributes
	To          string        // field name to unmarshal to rather than .Name
	Desc        string        // descriptive label above field
	Note        string        // text summary of field restrictions (Min, Max, Add, AddMin)
	Help        string        // instructions on how to use inputs of this type
	Context     string        // explains how the user's input will be used
	Type        string        // kind of widget - i.e., "dropdown", "text", "calendar", etc
	Placeholder string        // ghosted in prompt for the user, e.g. "Type here..."
	Default     string        // populate the field with this value unless .Text is present
	Text        string        // this is always user input - use .Default for default values
	Replace     string        // replace this field with a Field from replaceData
	Shared      string        // pull in a []Value with this key from sharedData
	Ref         string        // Ref names another field which supplies this field's values
	Wrap        string        // wrap in an HTML element named with the value of Wrap
	OnAdd       string        // function to be executed in place of addFieldInstance
	OnRemove    string        // function to be executed in place of removeFieldInstance
	ValueModify string        // function that transforms widget value on search submit
	ValueSet    string        // function that ignores widget value and sets its own
	Init        string        // key to map of server-side initialisation functions
	Data        FieldData     // data used to modify the appearance or functionality of widget
	Validate    []string      // name of validation functions for text input; e.g. hasLetter, etc
	Value       []Value       // the allowed range of values - if empty may signal user input
	RichText    RichText      // for rich text editor types only
	Field       []Field       // a field containing Field is treated as a subgroup
	Instances   []Field       // instances of this field, for populating editor
	Events      Events

	/*
		Min/Max differ in their precise meaning based on the field's
		.Type value. For text/textarea types it refers to runes. For
		tagger types it's the number of tags. For files it's how many
		kilobytes an individual file may be.
	*/
	Min int // users must add at least this many values in the field
	Max int // users may add up to this many values in the field

	Add    int // users may add up to this many instances of the field
	AddMin int // users must add at least this many instances of the field

	Percent int // similar to Paired but allows arbitrary widths in percentages

	SubmitSingle  bool // allow single subgroups to be submitted
	NoGroupFormat bool // widgets in groupings will not be styled differently
	NoWrap        bool // widgets in groupings will not wrap
	Optional      bool // user does not have to fill out this field
	Paired        bool // pair beside other flagged fields if space allows
	HasIcons      bool // used to determine the presentation of dropdown menus
	Disabled      bool // sets the field/subgroup to be initially disabled
	AdminOnly     bool // field only accessible to admins; all AdminOnly fields are optional
	RequestOnly   bool // omitted from form initially - it may be requested by name
	ServerOnly    bool // omitted from form; cannot be requested & is only used for validation
	Hidden        bool // for hidden fields the user can't see that are auto-populated
	Dangerous     bool // activating this field deletes something
}

type FieldData []string

func (fd FieldData) Arg(i int) template.HTML {
	if len(fd) < i+1 {
		return ""
	}
	return template.HTML(fd[i])
}

type Evt struct {
	Handler string
	Type    string
	Args    []string
	Before  bool
}
type Events []Evt

func (ee Events) BeforeHandlers() template.HTML {
	var handlers []string
	for _, e := range ee {
		if e.Before {
			handlers = append(handlers, e.Handler)
		}
	}
	return template.HTML(strings.Join(handlers, ", "))
}
func (ee Events) BeforeTypes() template.HTML {
	var types []string
	for _, e := range ee {
		if e.Before {
			types = append(types, e.Type)
		}
	}
	return template.HTML(strings.Join(types, ", "))
}
func (ee Events) AfterHandlers() template.HTML {
	var handlers []string
	for _, e := range ee {
		if !e.Before {
			handlers = append(handlers, e.Handler)
		}
	}
	return template.HTML(strings.Join(handlers, ", "))
}
func (ee Events) AfterTypes() template.HTML {
	var types []string
	for _, e := range ee {
		if !e.Before {
			types = append(types, e.Type)
		}
	}
	return template.HTML(strings.Join(types, ", "))
}

type ValueData struct {
	Data  string
	Field string
}
type Value struct {
	Idx       int    // used for sorting
	Name      string // what's actually entered into the database, if present
	Data      string
	Text      string
	Desc      string
	Href      template.HTML
	Icon      template.HTML
	Default   bool
	True      bool    // this represents the truth value of a bool type
	Value     []Value // nested values are for things like multi-level menus
	ValueData []ValueData
}

type Button struct {
	Action    string
	Callback  string
	Dest      string
	Text      string
	Icon      template.HTML
	Dismiss   bool
	Dangerous bool
	Submit    bool
}

/*
	The html/template package treats all "data-" attributes
	as though they don't have that prefix. Since "data-action"
	becomes "action" (a form attritube that contains a URL for
	processing said form) it is assumed to have a URL in it and
	anything that makes that supposed URL ambiguous (which seems
	to include conditionals/variables) is disallowed.

	Hence the Actions method builds a singular string to get
	around this as we know that data-action will always contain
	the name of a client-side function, never a URL.
*/
func (b Button) Actions() template.HTML {
	var aa []string
	if b.Action != "" {
		aa = append(aa, b.Action)
	}
	if b.Submit {
		aa = append(aa, "submitModal")
	}
	if b.Dismiss {
		aa = append(aa, "dismissModal")
	}
	return template.HTML(strings.Join(aa, ","))
}
