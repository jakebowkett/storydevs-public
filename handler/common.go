package handler

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	sd "github.com/jakebowkett/storydevs"
)

func CacheControl(c *sd.Config, w http.ResponseWriter, sec float64, privacy string) {
	if !c.CacheControl {
		return
	}
	val := fmt.Sprintf("%s, max-age=%d", privacy, sec)
	w.Header().Add("Cache-Control", val)
}

func JSONResponse(
	w http.ResponseWriter,
	r *sd.Request,
	log sd.Logger,
	data interface{},
) {
	d, err := json.Marshal(data)
	if err != nil {
		log.BadRequest(r.Id, w, err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	Gzip(w, r, d, 200, log)
}

func Gzip(w http.ResponseWriter, r *sd.Request, p []byte, status int, log sd.Logger) {

	/*
		If the client doesn't accept GZIP encoding simply
		write the response with the default writer.
	*/
	enc := r.Request.Header.Get("Accept-Encoding")
	if !strings.Contains(enc, "gzip") {
		log.HttpStatus(r.Id, w, status)
		w.Write(p)
		return
	}

	// If no content type, apply sniffing algorithm to un-gzipped body.
	if "" == w.Header().Get("Content-Type") {
		w.Header().Set("Content-Type", http.DetectContentType(p))
	}

	w.Header().Set("Content-Encoding", "gzip")
	log.HttpStatus(r.Id, w, status)
	gz := gzip.NewWriter(w)
	defer gz.Close()
	gz.Write(p)
}

func HttpStatusText(r *sd.Request) string {
	status := http.StatusText(r.Status)
	status = strings.ToLower(status)
	status = strings.Replace(status, " ", "", -1)
	return status
}

func Base(c *sd.Config, cache sd.Cache, r *sd.Request) (*sd.Base, error) {

	css := cache.Load("css/styling.css")
	if css == nil {
		return nil, errors.New("couldn't find css")
	}

	js := cache.Load("js/init.js")
	if js == nil {
		return nil, errors.New("couldn't find js")
	}

	base := &sd.Base{
		Search: sd.Column{Name: "search"},
		Browse: sd.Column{Name: "browse", Empty: c.Empty["browse"]},
		Detail: sd.Column{Name: "detail", Empty: c.Empty["detail"]},
		Editor: sd.Column{Name: "editor", Empty: c.Empty["editor"]},

		MetaDesc:    c.SiteDesc,
		MetaCard:    c.SiteCardURL,
		MetaAlt:     c.SiteCardAlt,
		MetaURL:     r.Request.URL.String(),
		MetaTwitter: c.SiteTwitter,

		Styling:    css.CSS(),
		JavaScript: js.JS(),

		PresentThreshold: sd.PresentThreshold,
	}

	account, ok := r.User.(sd.Account)
	if ok {
		base.Account = account
	}

	return base, nil
}

func ExtractJson(w http.ResponseWriter, r *http.Request, body interface{}, maxLen int64) error {

	r.Body = http.MaxBytesReader(w, r.Body, maxLen)

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(data, &body)
	if err != nil {
		return err
	}

	return nil
}
