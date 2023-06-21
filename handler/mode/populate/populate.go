package populate

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	sd "github.com/jakebowkett/storydevs"
)

func Copy(mode, kind string, dep *sd.Dependencies, clearDefaults bool) (sd.Fields, error) {

	fn := fmt.Sprintf("ui/mode/%s_%s_init", mode, kind)
	ui := dep.Cache.Load(fn)
	if ui == nil {
		return nil, fmt.Errorf("couldn't find initialised %s %s UI", mode, kind)
	}

	f := struct {
		Search sd.Fields
		Editor sd.Fields
	}{}
	if err := toml.Unmarshal(ui.Bytes(), &f); err != nil {
		return nil, err
	}

	if kind == "editor" {
		return f.Editor, callUpdateFuncs(f.Editor, dep, clearDefaults)
	}
	return f.Search, callUpdateFuncs(f.Search, dep, clearDefaults)
}

func callUpdateFuncs(ff sd.Fields, dep *sd.Dependencies, clearDefaults bool) error {
	for i := range ff {
		f := &ff[i]
		if clearDefaults {
			f.Default = ""
		}
		if u, ok := dep.FieldUpdaters[f.Init]; ok {
			if err := u(f); err != nil {
				return err
			}
		}
		if len(f.Field) > 0 {
			if err := callUpdateFuncs(f.Field, dep, clearDefaults); err != nil {
				return err
			}
		}
	}
	return nil
}

func Editor(ed sd.Fields, mode string, cache sd.Cache, r sd.Resource) error {
	if err := populate(r, ed); err != nil {
		return err
	}
	return nil
}

func populate(r sd.Resource, mg sd.Fields) error {
	rv := reflect.ValueOf(r)
	p := &populater{mg: mg}
	return p.populate(rv)
}

type populater struct {
	path       []string
	mg         sd.Fields
	namedGroup bool
}

func (p *populater) current() string {
	path := strings.ToLower(strings.Join(p.path, "."))
	if !p.namedGroup {
		path = "*." + path
	}
	return path
}
func (p *populater) push(s string) {
	p.path = append(p.path, s)
}
func (p *populater) pop() {
	p.path = p.path[0 : len(p.path)-1]
}

func (p *populater) populate(rv reflect.Value) error {
	switch sd.ReflectKind(rv) {
	case reflect.Interface, reflect.Ptr:
		rv = rv.Elem()
		fallthrough
	case reflect.Struct:
		return p.populateStruct(rv)
	case reflect.Slice:
		return p.populateSlice(rv)
	case reflect.String:
		return p.populateString(rv)
	case reflect.Int, reflect.Int64:
		return p.populateInt(rv)
	case reflect.Bool:
		return p.populateBool(rv)
	}
	return errors.New("Unable to find field to set.")
}

func (p *populater) populateDateTime(rv reflect.Value) error {

	f, err := p.mg.Field(p.current())
	if err != nil {
		return err
	}
	if len(f.Field) != 2 {
		return errors.New("populate: expected sd.DateTime to have exactly 2 fields")
	}
	d := f.Field[0]
	t := f.Field[1]
	if d.Type != "date" {
		return errors.New("populate: expected first sd.DateTime field to be type date")

	}
	if t.Type != "time" {
		return errors.New("populate: expected second sd.DateTime field to be type time")
	}

	dt := rv.Interface().(sd.DateTime)
	if dt.Null {
		return nil
	}

	/*
		As dt.DateTime is in UTC we add its offset here. Time
		and date related widgets do not know about timezones.
	*/
	n := dt.DateTime + int64(dt.TZOff)
	secInDay := int64(((1 * 60) * 60) * 24)

	p.push(d.Name)
	p.mg.SetWithInt(p.current(), n)
	p.pop()

	p.push(t.Name)
	p.mg.SetWithInt(p.current(), n%secInDay)
	p.pop()

	return p.mg.SetWithInt(p.current(), n)
}

func (p *populater) populateRange(rv reflect.Value) error {
	start := rv.Field(0).Interface().(string)
	end := rv.Field(1).Interface().(string)
	return p.mg.SetWithString(p.current(), start+"-"+end)
}

func (p *populater) populateFile(rv reflect.Value) error {
	f := rv.Interface().(sd.File)
	return p.mg.SetWithString(p.current(), f.Name.URL())
}

func (p *populater) populateStruct(rv reflect.Value) error {

	switch rv.Type().Name() {
	case "DateTime":
		return p.populateDateTime(rv)
	case "Range":
		return p.populateRange(rv)
	case "File":
		return p.populateFile(rv)
	}

	for i := 0; i < rv.NumField(); i++ {

		f := rv.Field(i)
		t := rv.Type().Field(i)
		name := t.Name

		if t.Tag.Get("ed") == "ignore" {
			continue
		}

		if tag := t.Tag.Get("ed_ref"); tag != "" {
			ref := rv.FieldByName(tag)
			name = ref.Interface().(string)
		}
		if tag := t.Tag.Get("ed_rm"); tag != "" {
			rm := p.current() + "." + tag
			if err := p.mg.RemoveField(rm); err != nil {
				return err
			}
		}
		if tag := t.Tag.Get("ed_wrap"); tag != "" {
			wrap := p.current() + "." + name
			if err := p.mg.WrapField(wrap, tag); err != nil {
				return err
			}
		}

		atBase := name == "ResourceBase"
		if !atBase {
			p.push(name)
		}
		if err := p.populate(f); err != nil {
			return err
		}
		if !atBase {
			p.pop()
		}
	}

	return nil
}

func (p *populater) populateSlice(rv reflect.Value) error {

	t := rv.Type()
	if t.Name() == "RichText" {
		f, err := p.mg.Field(p.current())
		if err != nil {
			return err
		}
		f.RichText = rv.Interface().(sd.RichText)
		return nil
	}

	kind := t.Elem().Kind()

	// String slices.
	if kind == reflect.String {
		var ss []string
		for i := 0; i < rv.Len(); i++ {
			e := rv.Index(i)
			ss = append(ss, e.Interface().(string))
		}
		if err := p.mg.SetWithSlice(p.current(), ss); err != nil {
			return err
		}
		return nil
	}

	// Struct slices.
	for i := 0; i < rv.Len(); i++ {

		p.namedGroup = true

		if err := p.mg.AddFieldInstance(p.current()); err != nil {
			return err
		}

		e := rv.Index(i)
		p.push(strconv.Itoa(i))
		if err := p.populate(e); err != nil {
			return err
		}
		p.pop()

		p.namedGroup = false
	}

	return nil
}

func (p *populater) populateString(rv reflect.Value) error {

	itf := rv.Interface()
	var s string
	switch itf.(type) {
	case string:
		s = itf.(string)
	case sd.NullString:
		s = itf.(sd.NullString).String
	case sd.FileName:
		s = itf.(sd.FileName).URL()
	default:
		return errors.New("unknown string type")
	}

	return p.mg.SetWithString(p.current(), s)
}

func (p *populater) populateInt(rv reflect.Value) error {

	itf := rv.Interface()
	var n int64
	switch itf.(type) {
	case int:
		n = int64(itf.(int))
	case int64:
		n = itf.(int64)
	case sd.NullInt64:
		n = itf.(sd.NullInt64).Int64
	default:
		return errors.New("unknown int type")
	}

	return p.mg.SetWithInt(p.current(), n)
}

func (p *populater) populateBool(rv reflect.Value) error {

	itf := rv.Interface()
	var b bool
	switch itf.(type) {
	case bool:
		b = itf.(bool)
	case sd.NullBool:
		b = itf.(sd.NullBool).Bool
	default:
		return errors.New("unknown bool type")
	}

	return p.mg.SetWithBool(p.current(), b)
}
