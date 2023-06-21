package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	sd "github.com/jakebowkett/storydevs"
)

type event struct {
	Name     string   `json:"name"`
	Summary  string   `json:"summary"`
	Start    int64    `json:"start"`
	Finish   int64    `json:"finish,omitempty"`
	Weekly   bool     `json:"weekly,omitempty"`
	Local    bool     `json:"local,omitempty"`
	Category []string `json:"category,omitempty"`
	Setting  []string `json:"setting,omitempty"`
}

func Event(dep *sd.Dependencies) sd.Handler {
	return func(w http.ResponseWriter, r *sd.Request) {
		handler(dep, w, r)
	}
}

func handler(dep *sd.Dependencies, w http.ResponseWriter, r *sd.Request) {

	log := dep.Logger
	db := dep.Resources["event"]

	res, err := db.Retrieve(r.Id, r.Vars["resource"], sd.ResOpts{})
	if err != nil {
		log.BadRequest(r.Id, w, err.Error())
		return
	}

	e, ok := res.(*sd.Event)
	if !ok {
		log.BadRequest(r.Id, w, "expected resource to be of type *sd.Event")
		return
	}

	summary := ""
	if e.Summary.String == "" {
		summary += e.GenerateSummary()
	} else {
		summary += e.Summary.String
	}

	category, err := mapValues(dep, "kind.category", e.Category)
	if err != nil {
		log.BadRequest(r.Id, w, err.Error())
		return
	}
	setting, err := mapValues(dep, "kind.setting", e.Setting)
	if err != nil {
		log.BadRequest(r.Id, w, err.Error())
		return
	}

	bb, err := json.Marshal(event{
		Name:     e.Name.String,
		Summary:  summary,
		Start:    e.Start.DateTime,
		Finish:   e.Finish.DateTime,
		Weekly:   e.Weekly.Bool,
		Local:    e.Timezone == "local",
		Category: category,
		Setting:  setting,
	})
	if err != nil {
		log.BadRequest(r.Id, w, err.Error())
		return
	}

	w.Write(bb)
}

func mapValues(
	dep *sd.Dependencies,
	fieldName string,
	names []string,
) (
	mapped []string,
	err error,
) {
	f, err := dep.ViewData.Mode["event"].Editor.Field(fieldName)
	if err != nil {
		return nil, err
	}
	for _, name := range names {
		text, ok := textFromName(f.Value, name)
		if !ok {
			return nil, fmt.Errorf("unable to map %q value %q", fieldName, name)
		}
		mapped = append(mapped, text)
	}
	return mapped, nil
}

func textFromName(vv []sd.Value, name string) (text string, ok bool) {
	for _, v := range vv {
		if v.Name == name {
			return v.Text, true
		}
	}
	return "", false
}
