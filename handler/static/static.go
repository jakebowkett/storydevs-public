package static

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	sd "github.com/jakebowkett/storydevs"
	"github.com/jakebowkett/storydevs/handler"
)

var isBase62 = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
var isFileName = regexp.MustCompile(`^[a-zA-Z0-9_]+\.[a-zA-Z0-9]+`)

func User(dep *sd.Dependencies) sd.Handler {

	c := dep.Config
	log := dep.Logger
	db := dep.Db

	return func(w http.ResponseWriter, r *sd.Request) {

		file := r.Vars["file"]

		// Cookies aren't necessarily required to
		// retrieve files so we ignore the error.
		var token string
		cookie, _ := r.Request.Cookie("token")
		if cookie != nil {
			token = cookie.Value
		}

		if !isFileName.MatchString(file) {
			log.NotFound(r.Id, w)
			return
		}

		/*
			This query tests whether the resource
			this file belongs to has:
				1.) non-private visibility OR
				2.) is owned by the client OR
				3.) if the account accessing it is an admin

			There is a bug of sorts with this query, though it's
			manageable. If any of the tables are completely empty
			the file will be reported as not existing. I don't
			understand why yet and don't care. Just make sure they
			contain at least one entry.
		*/
		exists, err := db.Exists(`
			FROM
		        logins,
		        personas,
		        profile,
		        post,
		        event,
		        file
			WHERE
			    file.file = $1 AND
			    (
					(
						logins.token = $2 AND
						logins.acc_id = personas.acc_id AND
						personas.admin = true
					) OR (
				    	(
				    		file.persona = personas.id AND
				    		(
				    			personas.visibility != 'private' OR
				    			(
						    		personas.acc_id = logins.acc_id AND
						    		logins.token = $2
				    			)
				    		)
				    	)

				    	OR

				    	(
				    		file.profile = profile.id AND
				    		(
				    			profile.visibility != 'private' OR
				    			(
						    		profile.ref_id = personas.id AND
						    		personas.acc_id = logins.acc_id AND
						    		logins.token = $2
				    			)
				    		)
				    	)

				    	OR

				    	(
				    		file.post = post.id AND
				    		(
				    			post.visibility != 'private' OR
				    			(
						    		post.ref_id = personas.id AND
						    		personas.acc_id = logins.acc_id AND
						    		logins.token = $2
				    			)
				    		)
				    	)

				    	OR

				    	(
				    		file.event = event.id AND
				    		(
				    			event.visibility != 'private' OR
				    			(
						    		event.ref_id = personas.id AND
						    		personas.acc_id = logins.acc_id AND
						    		logins.token = $2
				    			)
				    		)
				    	)
		    		)
				)`,
			file, token)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}
		if !exists {
			log.NotFound(r.Id, w)
			return
		}

		path, err := filepath.Abs(c.DirUser + "/" + file)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}
		f, err := os.Open(path)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}
		defer f.Close()
		fi, err := f.Stat()
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}

		handler.CacheControl(c, w, c.CacheUserFiles.Seconds(), sd.CachePrivate)
		http.ServeContent(
			w,
			r.Request,
			r.Request.URL.Path,
			fi.ModTime(),
			f,
		)
	}
}

func Media(dep *sd.Dependencies) sd.Handler {

	c := dep.Config
	log := dep.Logger
	cache := dep.Cache

	return func(w http.ResponseWriter, r *sd.Request) {

		path := r.Request.URL.Path

		if path == "/favicon.ico" {
			handler.CacheControl(c, w, c.CacheFavicon.Seconds(), sd.CachePublic)
		} else {
			handler.CacheControl(c, w, c.CacheSiteFiles.Seconds(), sd.CachePublic)
		}

		obj := cache.Load(path[1:])
		if obj == nil {
			log.NotFound(r.Id, w)
			return
		}

		http.ServeContent(
			w,
			r.Request,
			path,
			obj.LastMod(),
			bytes.NewReader(obj.Bytes()),
		)
	}
}

func Robots(dep *sd.Dependencies) sd.Handler {

	c := dep.Config
	log := dep.Logger
	cache := dep.Cache

	return func(w http.ResponseWriter, r *sd.Request) {

		obj := cache.Load("robots.txt")
		if obj == nil {
			log.NotFound(r.Id, w)
			return
		}

		handler.CacheControl(c, w, c.CacheRobots.Seconds(), sd.CachePublic)
		http.ServeContent(
			w,
			r.Request,
			r.Request.URL.Path,
			obj.LastMod(),
			bytes.NewReader(obj.Bytes()),
		)
	}
}

func Handler(dep *sd.Dependencies) sd.Handler {

	log := dep.Logger
	cache := dep.Cache

	return func(w http.ResponseWriter, r *sd.Request) {

		path := r.Request.URL.Path

		obj := cache.Load(path[1:])
		if obj == nil {
			log.NotFound(r.Id, w)
			return
		}

		var rs io.ReadSeeker
		var err error
		enc := r.Request.Header.Get("Accept-Encoding")
		if strings.Contains(enc, "gzip") {
			w.Header().Set("Content-Encoding", "gzip")
			rs, err = gzipBytes(obj.Bytes())
		} else {
			rs = bytes.NewReader(obj.Bytes())
		}
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}

		/*
			We set this header to prevent leaking the "Referer" [sic]
			header (whose URL may contain sensitive information such as
			a reset password token) via CSS files that are requesting
			resources.

			The base.html template contains the same setting and it applies
			to CSS contained directly in the rendered templates but not to
			externally loaded CSS files like those potentially served here.
		*/
		w.Header().Set("Referrer-Policy", "no-referrer")

		// handler.CacheControl(c, w, c.CacheSiteFiles.Seconds(), sd.CachePublic)
		http.ServeContent(
			w,
			r.Request,
			path,
			obj.LastMod(),
			rs,
		)
	}
}

func gzipBytes(p []byte) (io.ReadSeeker, error) {

	r, w := io.Pipe()
	defer r.Close()
	defer w.Close()

	buf := new(bytes.Buffer)

	go func() {
		buf.ReadFrom(r)
	}()

	gz := gzip.NewWriter(w)
	_, err := gz.Write(p)
	if err != nil {
		return nil, err
	}
	gz.Close()

	return bytes.NewReader(buf.Bytes()), nil
}
