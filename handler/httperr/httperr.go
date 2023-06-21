package httperr

import (
	"net/http"
	"strings"

	sd "github.com/jakebowkett/storydevs"
	"github.com/jakebowkett/storydevs/handler"
	"github.com/jakebowkett/storydevs/handler/page"
)

func Error(dep *sd.Dependencies) sd.Handler {

	log := dep.Logger
	vd := dep.ViewData

	return func(w http.ResponseWriter, r *sd.Request) {
		if r.Error != nil {
			log.Error(r.Id, r.Error.Error())
		}
		if isPartial(r) {
			errorPartial(w, r, log, vd)
		} else {
			r.Vars["page"] = "error"
			page.Full(dep)(w, r)
		}
	}
}

func errorPartial(
	w http.ResponseWriter,
	r *sd.Request,
	log sd.Logger,
	vd *sd.ViewData,
) {
	statusText := handler.HttpStatusText(r)
	p := []byte(vd.Errors[statusText])
	handler.Gzip(w, r, p, r.Status, log)
}

func isPartial(r *sd.Request) bool {
	if strings.HasSuffix(r.Request.URL.Path, "/partial") {
		return true
	}
	switch r.Request.Method {
	case "PUT", "POST", "DELETE":
		return true
	}
	return false
}
