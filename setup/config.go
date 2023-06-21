package setup

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"

	"github.com/BurntSushi/toml"
	sd "github.com/jakebowkett/storydevs"
)

func mustConfig(path string) *sd.Config {

	// Load default config.
	path, err := filepath.Abs(path)
	if err != nil {
		panic(err)
	}
	f, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	c := &sd.Config{}
	if err = toml.Unmarshal(f, c); err != nil {
		panic(err)
	}

	/*
		Load local config. Settings in the local config
		are meant to override the default config, hence
		we unmarshal into the same struct.
	*/
	f, err = ioutil.ReadFile(c.PathConfigLocal)
	if err != nil {
		panic(err)
	}
	if err = toml.Unmarshal(f, c); err != nil {
		panic(err)
	}

	// Load credentials and assign them to the appropriate field.
	f, err = ioutil.ReadFile(c.PathCredentials)
	if err != nil {
		panic(err)
	}
	var cred sd.Credentials
	if err = toml.Unmarshal(f, &cred); err != nil {
		panic(err)
	}
	c.Credentials = cred

	// Create any directories referenced by the config.
	rv := reflect.ValueOf(*c)
	rt := rv.Type()
	isDir := regexp.MustCompile(`^Dir[A-Z].*`)
	for i := 0; i < rt.NumField(); i++ {
		ft := rt.Field(i)
		if !isDir.MatchString(ft.Name) {
			continue
		}
		fv := rv.Field(i)
		dir := fv.Interface().(string)
		dir, err := filepath.Abs(dir)
		if err != nil {
			panic(err)
		}
		if err := os.MkdirAll(dir, 0700); err != nil {
			panic(fmt.Errorf("%w: unable to create directory: %s", err, dir))
		}
	}

	return c
}
