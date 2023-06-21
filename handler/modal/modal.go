package modal

import (
	"html/template"
	"net/http"

	sd "github.com/jakebowkett/storydevs"
	"github.com/jakebowkett/storydevs/handler"
)

func Full(dep *sd.Dependencies) sd.Handler {

	c := dep.Config
	log := dep.Logger
	view := dep.Templates
	cache := dep.Cache
	vd := dep.ViewData

	return func(w http.ResponseWriter, r *sd.Request) {

		modal, name, err := renderPartial(log, view, r, vd)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}

		base, err := handler.Base(c, cache, r)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}
		base.View = "home"
		base.ViewType = "page"
		base.ViewMeta = vd
		base.Layout = "page"
		base.Title = vd.Modal[name].Title
		base.Modal = template.HTML(modal)

		v, err := view.Render("base.html", base)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}

		// handler.CacheControl(c, w, c.CacheHTML.Seconds(), sd.CachePrivate)
		handler.Gzip(w, r, v, 200, log)
	}
}

func Partial(dep *sd.Dependencies) sd.Handler {

	log := dep.Logger
	view := dep.Templates
	vd := dep.ViewData

	return func(w http.ResponseWriter, r *sd.Request) {

		modal, _, err := renderPartial(log, view, r, vd)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}

		// handler.CacheControl(c, w, c.CacheHTML.Seconds(), sd.CachePublic)
		handler.Gzip(w, r, modal, 200, log)
	}
}

func renderPartial(
	log sd.Logger,
	view sd.View,
	r *sd.Request,
	vd *sd.ViewData,
) (
	[]byte,
	string,
	error,
) {

	name := r.Vars["modal"]
	data := vd.Modal[name]
	account, ok := r.User.(sd.Account)
	if ok {
		p := account.ActivePersona()
		data.IsAdmin = p.Admin.Bool
	}

	modal, err := view.Render("modal.html", data)
	if err != nil {
		return nil, name, err
	}

	return modal, name, nil
}
