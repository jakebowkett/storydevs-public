package mode

import (
	"net/http"
	"os"
	"path/filepath"

	sd "github.com/jakebowkett/storydevs"
	"github.com/jakebowkett/storydevs/handler"
	"github.com/jakebowkett/storydevs/handler/mode/submit"
)

func Delete(dep *sd.Dependencies) sd.Handler {

	log := dep.Logger
	c := dep.Config
	rs := dep.Resources

	return func(w http.ResponseWriter, r *sd.Request) {

		rSlug := idFromSlug(r.Vars["resource"])

		metaName := submit.MetaName(r)

		if r.User == nil {
			var lk string
			switch metaName {
			case "talent":
				lk = sd.LK_ProfileSlug
			case "library", "forums":
				lk = sd.LK_ThreadSlug
			default:
				log.BadRequest(r.Id, w, "Couldn't find slug log key for this mode.")
				return
			}
			log.BadRequest(r.Id, w, "No user while attempting to delete resource.").
				Data(lk, rSlug)
			return
		}

		user := r.User.(sd.Account)
		p := user.ActivePersona()

		// Delete associated database entries.
		fb, toRemove, err := rs[metaName].Delete(r.Id, rSlug, p.Id)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}

		if len(fb) > 0 {
			m := make(map[string]interface{})
			m["feedback"] = fb
			handler.JSONResponse(w, r, log, m)
			return
		}

		/*
			Remove files that are no longer needed by the
			resource. We log any error but we don't return
			because from the user's perspective nothing is
			wrong. It's a server problem that redundant files
			remain on disk.
		*/
		if err := removeFiles(c.DirUser, toRemove); err != nil {
			log.Error(r.Id, err.Error())
		}

		// Delete associated file directory.
		dir := filepath.Join(c.DirUser, p.Slug, metaName, rSlug)
		if err := os.RemoveAll(dir); err != nil {
			log.BadRequest(r.Id, w, err.Error())
		}
	}
}
