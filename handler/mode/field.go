package mode

import (
	"net/http"

	sd "github.com/jakebowkett/storydevs"
	"github.com/jakebowkett/storydevs/handler"
)

func Field(dep *sd.Dependencies) sd.Handler {

	log := dep.Logger
	view := dep.Templates
	mdMap := dep.ViewData.Mode

	return func(w http.ResponseWriter, r *sd.Request) {

		mode := r.Vars["mode"]
		section := r.Vars["section"]
		name := r.Vars["name"]
		account := r.User.(sd.Account)
		persona := account.ActivePersona()

		var f *sd.Field
		var err error

		switch section {
		case "search":
			f, err = mdMap[mode].Search.Field(name)
		case "editor":
			f, err = mdMap[mode].Editor.Field(name)
		}

		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}

		if f.ServerOnly {
			log.BadRequest(r.Id, w, "requested field is set to .ServerOnly")
			return
		}

		args := []interface{}{f, false, persona.Admin.Bool, 0, 0}
		p, err := view.Render("field", args)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}
		handler.Gzip(w, r, p, 200, log)
	}
}
