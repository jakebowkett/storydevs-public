package object

import (
	"errors"
	"reflect"
	"strconv"
	"strings"

	sd "github.com/jakebowkett/storydevs"
)

func Set(name string, value, obj interface{}) error {
	rv := reflect.ValueOf(obj)
	o := &object{
		name:  strings.Split(strings.ToLower(name), "."),
		value: value,
	}
	return o.find(rv)
}

type object struct {
	path  []string
	name  []string
	value interface{}
}

func (o *object) push(s string) {
	o.path = append(o.path, s)
}
func (o *object) pop() {
	o.path = o.path[0 : len(o.path)-1]
}
func (o *object) match() bool {
	if len(o.path) != len(o.name) {
		return false
	}
	for i := 0; i < len(o.path); i++ {
		if o.path[i] != o.name[i] {
			return false
		}
	}
	return true
}
func (o *object) couldMatch() bool {
	if len(o.path) > len(o.name) {
		return false
	}
	for i := 0; i < len(o.path); i++ {
		if o.path[i] != o.name[i] {
			return false
		}
	}
	return true
}

func (o *object) find(rv reflect.Value) error {
	switch rv.Kind() {
	case reflect.Interface, reflect.Ptr:
		rv = rv.Elem()
		fallthrough
	case reflect.Struct:
		return o.findField(rv)
	case reflect.Slice:
		return o.findIndex(rv)
	}
	return errors.New("object: unable to find field to set.")
}

func (o *object) findField(rv reflect.Value) error {

	for i := 0; i < rv.NumField(); i++ {

		rf := rv.Field(i)
		name := strings.ToLower(rv.Type().Field(i).Name)
		o.push(name)

		if o.match() {
			return o.setField(rf)
		}

		if !o.couldMatch() {
			o.pop()
			continue
		}

		return o.find(rf)
	}

	return errors.New("object: couldn't find field")
}

func (o *object) findIndex(rv reflect.Value) error {

	for i := 0; i < rv.Len(); i++ {

		elem := rv.Index(i)
		o.push(strconv.Itoa(i))

		if o.match() {
			return o.setIndex(elem)
		}

		if !o.couldMatch() {
			o.pop()
			continue
		}

		return o.find(elem)
	}

	return errors.New("object: couldn't find index")
}

func (o *object) setField(rv reflect.Value) error {

	typ := rv.Type()
	v, ok := o.value.(sd.ReadSeekCloser)
	if !ok {
		return errors.New("object: value is not of type sd.ReadSeekCloser")
	}

	if typ.Name() == "File" {
		d := rv.FieldByName("Data")
		d.Set(reflect.ValueOf(v))
		return nil
	}

	rsc := reflect.TypeOf((*sd.ReadSeekCloser)(nil)).Elem()
	if typ.Implements(rsc) {
		rv.Set(reflect.ValueOf(v))
		return nil
	}

	return errors.New("object: couldn't set field")
}

func (o *object) setIndex(rv reflect.Value) error {
	return errors.New("object: setIndex is not implemented yet")
}
