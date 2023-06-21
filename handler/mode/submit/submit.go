package submit

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	sd "github.com/jakebowkett/storydevs"
	"github.com/jakebowkett/storydevs/handler/mode/object"
)

func MetaName(r *sd.Request) string {
	if sm, ok := r.Vars["submode"]; ok {
		return sm
	}
	return r.Vars["mode"]
}

func ResourceInstance(metaName string) (resource sd.Resource, err error) {
	switch metaName {
	case "talent":
		resource = &sd.Profile{}
	case "event":
		resource = &sd.Event{}
	case "library":
		resource = &sd.Post{Kind: []string{"library"}}
	case "forums":
		resource = &sd.Post{Kind: []string{"forums"}}
	case "identity":
		resource = &sd.SettingsIdentity{}
	case "privacy":
		resource = &sd.SettingsPrivacy{}
	default:
		return nil, errors.New("submit: unhandled ResourceInstance case")
	}
	return resource, nil
}

func MultipartJSON(w http.ResponseWriter, r *http.Request, res sd.Resource, max int64) ([]namedCloser, error) {

	r.Body = http.MaxBytesReader(w, r.Body, max)

	err := r.ParseMultipartForm(max)
	if err != nil {
		return nil, err
	}

	j, ok := r.MultipartForm.Value["json"]
	if !ok {
		return nil, errors.New("Expected JSON field in multi-part form.")
	}
	if len(j) != 1 {
		return nil, errors.New(`Expected field "json" to have length of 1.`)
	}

	err = json.Unmarshal([]byte(j[0]), &res)
	if err != nil {
		return nil, err
	}

	var opened []namedCloser
	for name, ff := range r.MultipartForm.File {
		for _, fh := range ff {
			f, err := fh.Open()
			if err != nil {
				return opened, err
			}
			opened = append(opened, namedCloser{
				f.(io.Closer),
				fh.Filename,
			})
			if err := object.Set(name, f, res); err != nil {
				return opened, err
			}
		}
	}

	return opened, nil
}

func CloseFiles(reqId string, log sd.Logger, closers []namedCloser) (err error) {
	for _, c := range closers {
		if cErr := c.Close(); cErr != nil {
			log.ErrorF(reqId, "unable to close file %q: %w", c.Name, cErr)
			err = cErr
		}
	}
	return err
}

type namedCloser struct {
	io.Closer
	Name string
}

func RemoveNewFiles(paths []string) error {
	for i, path := range paths {
		if err := os.Remove(path); err != nil {
			remaining := strings.Join(paths[i:], ",\n")
			return fmt.Errorf("%w:\nunable to remove all new files:\n%s", err, remaining)
		}
	}
	return nil
}
