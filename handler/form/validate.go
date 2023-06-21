package form

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	sd "github.com/jakebowkett/storydevs"
	"github.com/jakebowkett/storydevs/handler/mode/submit"
)

func Validate(
	reqId string,
	c *sd.Config,
	log sd.Logger,
	view string,
	r sd.Resource,
	m map[string]string,
	ff sd.Fields,
	reply bool,
	retry sd.Tryer,
) (
	v *validator,
	err error,
) {
	/*
		Remove files written so far upon error. It is important
		that all return sites return v otherwise the deferred
		call will panic due to it being nil.
	*/
	defer func() {
		if err != nil {
			if rmErr := submit.RemoveNewFiles(v.TableTree.Written); rmErr != nil {
				err = fmt.Errorf("%w:\n%s", err, rmErr.Error())
			}
		}
	}()

	// General validation.
	v = &validator{
		admin:   r.GetAdmin(),
		pId:     r.OwnerId(),
		pSlug:   r.GetPersSlug(),
		name:    reflect.ValueOf(r).Elem().Type().Name(),
		handle:  r.GetHandle(),
		mapping: m,
		source:  ff,
		config:  c,
		log:     log,
		mode:    view,
		reply:   reply,
		retry:   retry,
	}
	if err = v.validate(reflect.ValueOf(r), nil, ignore{}); err != nil {
		return v, err
	}

	// Mode specific validation.
	switch view {
	case "talent":
		err = talent(r.(*sd.Profile), ff)
	}
	if err != nil {
		return v, err
	}

	return v, nil
}

type ignore struct {
	db         bool
	validation bool
}

type validator struct {
	reqId string

	config *sd.Config

	log sd.Logger

	// Name of resource type (used for creating db tables).
	name string

	// Mode name.
	mode string

	// True if replying to a resource, false otherwise.
	reply bool

	// Is the account an admin.
	admin bool

	// Persona id, slug & handle.
	pId    int64
	pSlug  string
	handle string

	// Resource slug.
	rSlug string

	path    []string
	mapping map[string]string
	source  sd.Fields

	retry sd.Tryer

	src string
	sf  *sd.Field

	// Tree structure of tables representing the DB layout.
	TableTree *sd.DbTable
}

func (v *validator) validate(rv reflect.Value, tbl *sd.DbTable, ignore ignore) (err error) {

	if err = v.currentSource(ignore); err != nil {
		return err
	}

	switch sd.ReflectKind(rv) {
	case reflect.Interface:
		break
	case reflect.Ptr:
		rv = rv.Elem()
		fallthrough
	case reflect.Struct:
		return v.validateStruct(rv, tbl, ignore)
	case reflect.Array, reflect.Slice:
		return v.validateSequence(rv, tbl, ignore)
	case reflect.String:
		return v.validateString(rv, tbl, ignore)
	case reflect.Int, reflect.Int64:
		return v.validateInt(rv, tbl, ignore)
	case reflect.Float32, reflect.Float64:
		return v.validateFloat(rv, tbl, ignore)
	case reflect.Bool:
		return v.validateBool(rv, tbl, ignore)
	}

	if !ignore.validation {
		return fmt.Errorf("unexpected kind %q at %q", rv.Kind().String(), v.src)
	}

	return nil
}

func (v *validator) addTable(tbl *sd.DbTable) *sd.DbTable {
	newTbl := &sd.DbTable{
		Name: v.currentDbTable(),
	}
	if tbl == nil {
		newTbl.Refs = v.pId
		v.TableTree = newTbl
	} else {
		tbl.Tables = append(tbl.Tables, newTbl)
	}
	return newTbl
}

func (v *validator) valToTable(tbl *sd.DbTable, val interface{}) {
	name := v.path[len(v.path)-1]
	tbl.Columns = append(tbl.Columns, strings.ToLower(name))
	tbl.Values = append(tbl.Values, val)
}

func (v *validator) validateStruct(rv reflect.Value, tbl *sd.DbTable, ignore ignore) error {

	structType := rv.Type().Name()

	switch structType {
	case "File":
		return v.validateFile(rv, tbl, ignore)
	case "Range":
		return v.validateRange(rv, tbl, ignore)
	case "DateTime":
		return v.validateInt(rv, tbl, ignore)
	}

	if structType != "ResourceBase" {
		tbl = v.addTable(tbl)
	}

	/*
		Media struct needs special attention, although
		some of its fields are still processed normally,
		hence we continue normal validation if there's
		no error, ignoring the fields of Media that are
		handled here.
	*/
	if structType == "Media" {
		if err := v.validateMedia(rv, tbl, ignore); err != nil {
			return err
		}
	}

	for i := 0; i < rv.NumField(); i++ {

		rf := rv.Field(i)
		f := rv.Type().Field(i)
		name := f.Name

		/*
			Make a copy of ignore local to the loop scope.
			The outer scope's value will be re-established
			each iteration.
		*/
		ignore := ignore
		if !ignore.validation {
			ignore.validation = f.Tag.Get("validate") == "ignore"
		}
		if !ignore.db {
			ignore.db = f.Tag.Get("database") == "ignore"
		}
		if ignore.db && ignore.validation {
			continue
		}
		if v.reply && f.Tag.Get("reply") == "ignore" {
			continue
		}

		atBase := name == "ResourceBase"
		if !atBase {
			v.pushPath(name)
		}
		if err := v.validate(rf, tbl, ignore); err != nil {
			return err
		}
		if !atBase {
			v.popPath()
		}
	}

	return nil
}

func (v *validator) validateRange(rv reflect.Value, tbl *sd.DbTable, ignore ignore) error {

	// Populate end with start value if end is empty.
	start := rv.Field(0).Interface().(string)
	end := rv.Field(1).Interface().(string)
	if end == "" {
		rv.Field(1).SetString(start)
	}

	var positions []int

	for i := 0; i < rv.NumField(); i++ {

		rf := rv.Field(i)
		f := rv.Type().Field(i)
		ignore := ignore
		ignore.db = true

		if err := v.validateString(rf, tbl, ignore); err != nil {
			return err
		}
		v.pushPath(v.sf.Name + "_" + f.Name)
		v.valToTable(tbl, rf.Interface())
		v.popPath()

		pos := -1
		for i, v := range v.sf.Value {
			if v.Name == rf.Interface().(string) {
				pos = i
				break
			}
		}
		if pos == -1 {
			return fmt.Errorf("Range type contains unknown %s value for field %q", f.Name, v.src)
		}

		positions = append(positions, pos)
	}

	if positions[0] > positions[1] {
		return fmt.Errorf("Range type has start greater than end for field %q", v.src)
	}

	return nil
}

func (v *validator) validateSequence(rv reflect.Value, tbl *sd.DbTable, ignore ignore) error {

	elemKind := rv.Type().Elem().Kind()
	length := rv.Len()

	if rv.Type().Name() == "RichText" {
		return v.validateRichText(rv, tbl, ignore)
	}

	if v.sf.Add != 0 {
		if length > v.sf.Add {
			msg := "Field %q exceeds maximum allowed elements. Max %d, got %d."
			return fmt.Errorf(msg, v.src, v.sf.Add, length)
		}
		if length < v.sf.AddMin {
			msg := "Field %q requires a minimum of %d elements, got %d."
			return fmt.Errorf(msg, v.src, v.sf.AddMin, length)
		}
	}

	if !v.sf.Optional && length == 0 {
		return fmt.Errorf("Non-optional field %q is zero length.", v.src)
	}

	for i := 0; i < length; i++ {
		elem := rv.Index(i)
		newTbl := tbl
		if elemKind != reflect.Struct {
			newTbl = v.addTable(tbl)
		}
		if err := v.validate(elem, newTbl, ignore); err != nil {
			return err
		}
	}

	return nil
}

func (v *validator) validateString(rv reflect.Value, tbl *sd.DbTable, ignore ignore) error {

	var s string
	var null bool
	itf := rv.Interface()

	switch itf.(type) {
	case string:
		s = itf.(string)
	case sd.NullString:
		ns := itf.(sd.NullString)
		s = ns.String
		null = ns.Null
	case sd.FileName:
		s = itf.(sd.FileName).String()
	default:
		return fmt.Errorf("Field %q contains unknown string type.", v.src)
	}

	s = strings.TrimSpace(s)

	if ignore.validation {
		goto end
	}

	if s == "" {
		if v.sf.Optional || v.sf.AdminOnly {
			if v.sf.Add > 0 {
				return fmt.Errorf("Optional field %q is a slice of strings that contains an empty string among its elements. An optional slice of strings should either contain no strings or non-empty strings.", v.src)
			}
			goto end
		} else {
			return fmt.Errorf("Non-optional field %q is empty.", v.src)
		}
	}
	if v.sf.AdminOnly && !v.admin {
		return fmt.Errorf("Admin only field %q set by non-admin account.")
	}
	if v.sf.Type == "textarea" {
		if ctrlCharSansNewline.MatchString(s) {
			return fmt.Errorf("Field %q contains non-newline control character(s).", v.src)
		}
	} else {
		if ctrlChar.MatchString(s) {
			return fmt.Errorf("Field %q contains control character(s).", v.src)
		}
	}
	if strLen(s) < v.sf.Min {
		return fmt.Errorf("Min rune count of field %q not met.", v.src)
	}
	if v.sf.Max > 0 && strLen(s) > v.sf.Max {
		return fmt.Errorf("Max rune count of field %q exceeded.", v.src)
	}

	for _, val := range v.sf.Validate {
		var re *regexp.Regexp
		switch val {
		case "isWord":
			re = isWord
		case "isEmail":
			re = isEmail
		case "isDiscord":
			re = isDiscord
		case "isDomain":
			re = isDomain
		default:
			return fmt.Errorf("Field %q contains unknown validation %q", v.src, val)
		}
		if !re.MatchString(s) {
			return fmt.Errorf("Field %q failed regexp %q", v.src, val)
		}
	}

	if checkSrcValue(v.sf) && !v.sf.InValue(s) {
		return fmt.Errorf("Field %q doesn't contain supplied value.", v.src)
	}

end:

	if !ignore.db {
		if null {
			v.valToTable(tbl, nil)
		} else {
			v.valToTable(tbl, s)
		}
	}

	return nil
}

func (v *validator) validateInt(rv reflect.Value, tbl *sd.DbTable, ignore ignore) error {

	var n int64
	var null bool
	itf := rv.Interface()

	switch itf.(type) {
	case int:
		n = int64(itf.(int))
	case int64:
		n = itf.(int64)
	case sd.NullInt64:
		ni := itf.(sd.NullInt64)
		n = ni.Int64
		null = ni.Null
	case sd.DateTime:
		dt := itf.(sd.DateTime)
		n = dt.DateTime
		null = dt.Null
	default:
		return fmt.Errorf("Field %q contains unknown integer type.", v.src)
	}

	if ignore.validation {
		goto end
	}

	if n == 0 && !v.sf.Optional {
		return fmt.Errorf("Non-optional field %q is zero.", v.src)
	}

end:

	if !ignore.db {
		if null {
			v.valToTable(tbl, nil)
		} else {
			v.valToTable(tbl, n)
		}
	}

	return nil
}

func (v *validator) validateFloat(rv reflect.Value, tbl *sd.DbTable, ignore ignore) error {

	var n float64
	var null bool
	itf := rv.Interface()

	switch itf.(type) {
	case float32:
		n = float64(itf.(float32))
	case float64:
		n = itf.(float64)
	case sd.NullFloat64:
		nf := itf.(sd.NullFloat64)
		n = nf.Float64
		null = nf.Null
	default:
		return fmt.Errorf("Field %q contains unknown float type.", v.src)
	}

	if ignore.validation {
		goto end
	}

	if n == 0 && !v.sf.Optional {
		return fmt.Errorf("Non-optional field %q is zero.", v.src)
	}

end:

	if !ignore.db {
		if null {
			v.valToTable(tbl, nil)
		} else {
			v.valToTable(tbl, n)
		}
	}

	return nil
}

func (v *validator) validateBool(rv reflect.Value, tbl *sd.DbTable, ignore ignore) error {

	var b bool
	var null bool
	itf := rv.Interface()

	switch itf.(type) {
	case bool:
		b = itf.(bool)
	case sd.NullBool:
		nb := itf.(sd.NullBool)
		b = nb.Bool
		null = nb.Null
		if v.sf.AdminOnly && !b {
			null = true
		}
	default:
		return fmt.Errorf("Field %q contains unknown bool type.", v.src)
	}

	if ignore.validation {
		goto end
	}

end:

	if !ignore.db {
		if null {
			v.valToTable(tbl, nil)
		} else {
			v.valToTable(tbl, b)
		}
	}

	return nil
}

func (v *validator) pushPath(s string) {
	v.path = append(v.path, s)
}
func (v *validator) popPath() {
	v.path = v.path[0 : len(v.path)-1]
}
func (v *validator) currentPath() string {
	return strings.ToLower(strings.Join(v.path, "."))
}

var isNum = regexp.MustCompile(`^\d+$`)

func (v *validator) currentDbTable() string {
	table := []string{v.name}
	for _, seg := range v.path {
		if isNum.MatchString(seg) {
			continue
		}
		table = append(table, seg)
	}
	return strings.ToLower(strings.Join(table, "_"))
}

func (v *validator) currentSource(ignore ignore) error {

	if ignore.validation {
		return nil
	}

	if len(v.path) == 0 {
		return nil
	}

	name := v.currentPath()
	src, ok := v.mapping[name]
	if !ok {
		return fmt.Errorf("Unable to map received field %q to source.", name)
	}

	sf, err := v.source.Field(src)
	if err != nil {
		return err
	}

	v.src = src
	v.sf = sf

	return nil
}

func checkSrcValue(sf *sd.Field) bool {
	if sf.Type == "range" {
		return true
	}
	if sf.Type == "dropdown" && len(sf.Value) > 0 {
		return true
	}
	if sf.Type == "checkbox" || sf.Type == "radio" {
		return true
	}
	return false
}

var (
	ctrlChar            = regexp.MustCompile(`\pC`)
	ctrlCharSansNewline = regexp.MustCompile(`[^\n\PC]`)
	isWord              = regexp.MustCompile(`^\w+$`)
	isEmail             = regexp.MustCompile(`^.+@.+\..+$`)
	isDiscord           = regexp.MustCompile(`^.+#\d{4}$`)
	isDomain            = regexp.MustCompile(`^.*[a-zA-Z0-9-]+\.[a-zA-Z0-9]+.*$`)
)

// var letterOrNumber = regexp.MustCompile(`[\pL|\d]`)

func strLen(s string) int {
	return len([]rune(s))
}
