package page

import (
	"encoding/json"
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

		page, name, err := renderPartial(r, dep)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}

		base, err := handler.Base(c, cache, r)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}
		base.View = name
		base.ViewType = "page"
		base.ViewMeta = vd
		base.Layout = "page"
		base.Title = vd.Page[name].Title
		base.Page = template.HTML(page)

		v, err := view.Render("base.html", base)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}

		var status int
		if r.Status >= 400 {
			status = r.Status
		} else {
			status = 200
		}

		// handler.CacheControl(c, w, c.CacheHTML.Seconds(), sd.CachePrivate)
		handler.Gzip(w, r, v, status, log)
	}
}

func Partial(
	dep *sd.Dependencies,
) sd.Handler {

	log := dep.Logger

	return func(w http.ResponseWriter, r *sd.Request) {

		page, _, err := renderPartial(r, dep)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}

		response := struct {
			Page template.HTML `json:"page"`
		}{
			Page: template.HTML(page),
		}

		p, err := json.Marshal(response)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}

		var status int
		if r.Status >= 400 {
			status = r.Status
		} else {
			status = 200
		}

		w.Header().Add("Content-Type", "application/json")
		// handler.CacheControl(c, w, c.CacheHTML.Seconds(), sd.CachePublic)
		handler.Gzip(w, r, p, status, log)
	}
}

func renderPartial(r *sd.Request, dep *sd.Dependencies) ([]byte, string, error) {

	view := dep.Templates
	vd := dep.ViewData
	rs := dep.Resources

	name, pd, err := populateData(r, vd, rs)
	if err != nil {
		return nil, name, err
	}

	page, err := view.Render(name+".html", pd)
	if err != nil {
		return nil, name, err
	}

	return page, name, nil
}

func populateData(r *sd.Request, vd *sd.ViewData, rs sd.Resources) (string, interface{}, error) {

	name := r.Vars["page"]
	if name == "" {
		name = "home"
	}

	if r.Status >= 400 {
		st := handler.HttpStatusText(r)
		pd := sd.PageData{
			Name:    st,
			Title:   http.StatusText(r.Status),
			Message: vd.Errors[st],
		}
		return name, pd, nil
	}

	return name, nil, nil
}
