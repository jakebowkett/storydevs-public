package mode

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	sd "github.com/jakebowkett/storydevs"
	"github.com/jakebowkett/storydevs/handler"
	"github.com/jakebowkett/storydevs/handler/form"
	"github.com/jakebowkett/storydevs/handler/mode/submit"
)

func Update(dep *sd.Dependencies) sd.Handler {

	c := dep.Config
	log := dep.Logger
	mp := dep.ResourceMapping
	md := dep.ViewData.Mode
	rs := dep.Resources
	view := dep.Templates
	retry := dep.TryerDisk

	return func(w http.ResponseWriter, r *sd.Request) {

		// Get the active persona.
		if r.User == nil {
			log.Unauthorised(r.Id, w)
			return
		}
		account := r.User.(sd.Account)
		persona := account.ActivePersona()

		// Get the mode name - this will be the
		// submode for /account or /admin paths.
		metaName := submit.MetaName(r)

		// Get an instance of this mode's resource.
		var iName string
		inSettings := metaName == "settings"
		if inSettings {
			iName = r.Vars["resource"]
		} else {
			iName = metaName
		}
		resource, err := submit.ResourceInstance(iName)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}
		now := time.Now().Unix()
		rSlug := idFromSlug(r.Vars["resource"])
		resource.SetUpdated(now)
		resource.SetOwner(persona)
		resource.SetSlug(rSlug)

		// Parse the multipart form and make sure to close the original files.
		max := c.MaxForm[metaName]
		openFiles, err := submit.MultipartJSON(w, r.Request, resource, max)
		defer submit.CloseFiles(r.Id, log, openFiles)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}

		// Detect whether this is a reply.
		reply := false
		t, threader := resource.(sd.Threader)
		if threader && t.GetThread() != "" {
			reply = true
		}
		if reply && !in([]string{"library", "forums"}, metaName) {
			log.BadRequest(r.Id, w, "Cannot reply to modes other than library or forums.")
			return
		}

		/*
			Validate resource and commit new files to disk. Any
			files that are committed prior to encountering an
			error will be removed by form.Validate
		*/
		mapping := mp[metaName]
		data := md[metaName]
		result, err := form.Validate(r.Id, c, log, metaName, resource, mapping, data.Editor, reply, retry)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}

		/*
			Forum threads/posts are always public. Since it's
			possible (though not supported by the UI) for the
			client to submit to the forums with *any* valid
			visibility setting we manually set it here to
			override any potential sketchiness.
		*/
		if metaName == "forums" {
			result.TableTree.SafeAdd("visibility", sd.VisibilityPublic)
		}

		/*
			During an update the TableTree will need the
			resource slug so that insertTables will know
			which root table to update.
		*/
		result.TableTree.Slug = rSlug

		/*
			Commit the updated resource to database. If
			the DB commit fails we remove the new files.
		*/
		fb, toRemove, err := rs[metaName].Update(r.Id, resource, result.TableTree)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			if err := submit.RemoveNewFiles(result.TableTree.Written); err != nil {
				log.Error(r.Id, err.Error())
			}
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

		/*
			If the user submitted a reply to a thread we return
			the whole thread not just the user's reply. Therefore
			resource will be a thread containing the reply that
			was submitted nested somewhere in it.
		*/
		if threader {
			mode := r.Vars["mode"]
			getPrivate := mode == "account" || mode == "admin"
			slug := resource.(sd.Threader).GetThread()
			resource, err = rs[metaName].Retrieve(r.Id, slug, sd.ResOpts{
				GetPrivate: getPrivate,
			})
			if err != nil {
				log.BadRequest(r.Id, w, err.Error())
				return
			}
		}

		if inSettings {
			w.WriteHeader(http.StatusCreated)
			return
		}

		// Respond with rendered resource.
		data.Name = metaName
		data.IsAdmin = persona.Admin.Bool
		data.Account = r.User.(sd.Account)
		data.InAccount = r.Vars["mode"] == "account"
		data.InAdmin = r.Vars["mode"] == "admin"
		data.Resource = resource
		v, err := view.Render("detail.html", data)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}

		// Respond with JSON payload containing resource and its slug.
		m := make(map[string]interface{})
		m["resource"] = string(v)
		m["slug"] = resource.GetSlug()
		if reply {
			m["reply"] = rSlug
		}
		j, err := json.Marshal(m)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}
		w.Header().Set("Content-Type", "application/json")
		handler.Gzip(w, r, j, http.StatusCreated, log)
	}
}

func removeFiles(dir string, toRemove []string) error {
	for i, f := range toRemove {
		f, err := filepath.Abs(filepath.Join(dir, f))
		if err != nil {
			return removeErr(err, toRemove, i)
		}
		if err = os.Remove(f); err != nil {
			return removeErr(err, toRemove, i)
		}
	}
	return nil
}

func removeErr(err error, toRemove []string, i int) error {
	remaining := strings.Join(toRemove[i:], ",\n")
	return fmt.Errorf("%w: couldn't remove files:\n"+remaining, err)
}
