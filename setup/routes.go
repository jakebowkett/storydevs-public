package setup

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	sd "github.com/jakebowkett/storydevs"
	"github.com/jakebowkett/storydevs/handler/account"
	"github.com/jakebowkett/storydevs/handler/api"
	"github.com/jakebowkett/storydevs/handler/httperr"
	"github.com/jakebowkett/storydevs/handler/modal"
	"github.com/jakebowkett/storydevs/handler/mode"
	"github.com/jakebowkett/storydevs/handler/page"
	"github.com/jakebowkett/storydevs/handler/static"
	"github.com/jakebowkett/storydevs/internal/router"
)

func MustRoutes(dep *sd.Dependencies) *router.Router {

	ms := dep.Modals
	log := dep.Logger

	// Set up handlers.
	o := router.Options{
		Error:    httperr.Error(dep),
		Recover:  recoverHandler(dep),
		Deferred: DeferredRequest(dep),
	}
	rt := router.New(o)

	rt.Use(beforeHandler(dep))

	/* =================================================
	   | API                                           |
	   ============================================== */
	rt.Get("/api/event/:resource", api.Event(dep))

	mediaHandler := static.Media(dep)
	rt.Get("/favicon.ico", mediaHandler)
	rt.Get("/robots.txt", static.Robots(dep))
	rt.Get("/[gfx,fonts]/:file", mediaHandler)

	/*
		User file requests incur a DB hit to check if
		they're private. We do this here before adding
		the account to avoid two hits for one resource.
	*/
	rt.Get("/user/:file", static.User(dep))

	mm := "talent,forums,event"
	mmAcc := "settings"

	// For gzipped site content.
	rt.Get("/[svg,js,css]/:file", static.Handler(dep))

	// Middleware that adds user to request object.
	rt.Use(account.Add(dep))

	/* =================================================
	   | Pages                                         |
	   ============================================== */

	pages := ""
	rt.Get("/:page[,"+pages+"]", page.Full(dep))
	rt.Get("/:page[home,"+pages+"]/partial", page.Partial(dep))

	/* =================================================
	   | Modals                                        |
	   ============================================== */

	modalFull := modal.Full(dep)
	modalPartial := modal.Partial(dep)
	modals := "/:modal[register,login,forgot,reserve,mailing]"

	rt.Get(modals, modalFull)
	rt.Get(modals+"/partial", modalPartial)

	rt.Pst("/register", ms.Register)
	rt.Pst("/login", ms.Login)
	rt.Pst("/forgot", ms.ForgotPassword)
	rt.Pst("/mailing", ms.Mailing)
	rt.Pst("/reserve", ms.Reserve)

	confirms := "/:kind[register,forgot,reserve,mailing,email,password]"
	rt.Pst(confirms+"/confirm", ms.ConfirmPartial)

	// Technically the modals directly under this comment should only be
	// accessible with an account. However, because of the "/:code" below
	// we cannot put them any lower.
	accModals := "/:modal[delete,email,password,persona,delete_account]"
	rt.Get(accModals, modalFull)
	rt.Get(accModals+"/partial", modalPartial)

	// Email links. These are placed below modalPartial handlers
	// so that a path like "/register/partial" isn't mistakenly
	// interpreted as a user's email confirmation code.
	rt.Get(confirms+"/:code", ms.ConfirmFull)
	rt.Get("/:kind[reserve_handle]/:code", ms.ConfirmFull) // legacy handle reservations

	/* =================================================
	   | Modes                                         |
	   ============================================== */

	modes := "/:mode" + "[" + mm + "]"
	submodes := "/:submode" + "[" + mm + "," + mmAcc + "]"
	modeFull := mode.Full(dep)
	modePartial := mode.Partial(dep)

	rt.Get(modes, modeFull)
	rt.Get(modes+"/partial", modePartial)
	rt.Get(modes+"/:resource", modeFull)
	rt.Get(modes+"/:resource/partial", modePartial)

	/* =================================================
	   | Account                                       |
	   ============================================== */

	acc := rt.Group("", nil, account.HasNone)

	acc.Put("/switch/:persona", account.Switch(dep))
	acc.Del("/persona/:persona", account.DeletePersona(dep))

	// These end points require an account.
	acc.Del("/logout", account.Logout(dep))
	acc.Pst("/email", ms.Email)
	acc.Pst("/password", ms.Password)
	acc.Pst("/persona", ms.Persona)
	acc.Del("/delete_account", ms.DeleteAccount)

	modeCreate := mode.Create(dep)
	modeUpdate := mode.Update(dep)
	modeDelete := mode.Delete(dep)

	rr := "forums"
	repliers := "/:mode[" + rr + "]"
	subRepliers := "/:submode[" + rr + "]"

	// Resource management.
	acc.Get(modes+"/:resource/edit", modeFull)
	acc.Get(modes+"/:resource/edit/partial", modePartial)
	acc.Get(repliers+"/:resource/reply", modeFull)
	acc.Get(repliers+"/:resource/reply/partial", modePartial)
	acc.Put(modes, modeCreate)
	acc.Put(modes+"/:resource", modeUpdate)
	acc.Del(modes+"/:resource", modeDelete)
	acc.Get(modes+"/:section[search,editor]/field/:name/partial", mode.Field(dep))

	/*
		Since account settings are not resources that can
		be created or deleted we place dummy handlers here
		prior to the general create/update/delete.
	*/
	acc.Put("/account/settings", account.CreateSettings)
	acc.Del("/account/settings/:resource", account.DeleteSettings)

	// Account access.
	acc.Get("/:mode[account]", modeFull)
	acc.Get("/:mode[account]/partial", modePartial)
	acc.Get("/:mode[account]"+submodes, modeFull)
	acc.Get("/:mode[account]"+submodes+"/partial", modePartial)
	acc.Get("/:mode[account]"+submodes+"/:resource", modeFull)
	acc.Get("/:mode[account]"+submodes+"/:resource/partial", modePartial)
	acc.Get("/:mode[account]"+subRepliers+"/:resource/reply", modeFull)
	acc.Get("/:mode[account]"+subRepliers+"/:resource/reply/partial", modePartial)
	acc.Get("/:mode[account]"+submodes+"/:resource/edit", modeFull)
	acc.Get("/:mode[account]"+submodes+"/:resource/edit/partial", modePartial)
	acc.Put("/:mode[account]"+submodes, modeCreate)
	acc.Put("/:mode[account]"+submodes+"/:resource", modeUpdate)
	acc.Del("/:mode[account]"+submodes+"/:resource", modeDelete)

	/* ==============================================
	   | Admin                                      |
	   ============================================== */

	adm := rt.Group("", account.NotAdmin, nil)
	adminSubs := "/:submode[" + mm + "]"
	adm.Get("/:mode[admin]", modeFull)
	adm.Get("/:mode[admin]/partial", modePartial)
	adm.Get("/:mode[admin]"+adminSubs, modeFull)
	adm.Get("/:mode[admin]"+adminSubs+"/partial", modePartial)
	adm.Get("/:mode[admin]"+adminSubs+"/:resource", modeFull)
	adm.Get("/:mode[admin]"+adminSubs+"/:resource/partial", modePartial)
	adm.Get("/:mode[admin]"+adminSubs+"/:resource/edit", modeFull)
	adm.Get("/:mode[admin]"+adminSubs+"/:resource/edit/partial", modePartial)
	adm.Put("/:mode[admin]"+adminSubs, modeCreate)
	adm.Put("/:mode[admin]"+adminSubs+"/:resource", modeUpdate)
	adm.Del("/:mode[admin]"+adminSubs+"/:resource", modeDelete)

	sess := log.Sess("Router setup.")
	for _, err := range rt.Errors {
		sess.Error(err.Error())
	}
	sess.End()

	if len(rt.Errors) > 0 {
		panic("invalid router patterns encountered")
	}

	return rt
}

func DeferredRequest(dep *sd.Dependencies) sd.Handler {
	log := dep.Logger
	return func(w http.ResponseWriter, r *sd.Request) {
		d := time.Since(r.Began)
		q := r.Request.URL.RawQuery
		if q != "" {
			q = "?" + q
		}
		log.End(
			r.Id,
			r.Request.RemoteAddr,
			r.Request.Method,
			r.Request.URL.Path+q,
			d.Nanoseconds(),
		)
	}
}

func recoverHandler(dep *sd.Dependencies) sd.Handler {
	log := dep.Logger
	return func(w http.ResponseWriter, r *sd.Request) {
		// When printing the panic we slice off everything before the root cause.
		ss := strings.Split(string(debug.Stack()), "\n")
		s := strings.Join(ss[9:], "\n")
		msg := fmt.Sprintf("Recovered from panic:\n%s\n%s", r.Error, s)
		log.BadRequest(r.Id, w, msg)
	}
}

func beforeHandler(dep *sd.Dependencies) sd.Handler {

	c := dep.Config
	log := dep.Logger

	var host string
	var scheme string
	if c.Dev {
		// host = "localhost:" + c.DevPort
		host = "localhost:" + c.Port
		scheme = "http"
	} else {
		host = "storydevs.com"
		scheme = "https"
	}

	// blacklistedIP := sd.NewTimedSet(time.Minute * 20)
	// blacklistedPath := sd.NewTimedSet(0)
	validFirstPathSegment := couldBeStoryDevsPath()

	return func(w http.ResponseWriter, r *sd.Request) {

		req := r.Request

		// if blacklistedIP.Has(req.RemoteAddr) {
		// 	code := http.StatusBadRequest
		// 	log.HttpStatus(r.Id, w, code)
		// 	log.Info(r.Id, "Terminating request from blacklisted IP address.")
		// 	r.Status = code
		// 	return
		// }

		// if blacklistedPath.Has(req.URL.Path) {
		// 	code := http.StatusBadRequest
		// 	log.HttpStatus(r.Id, w, code)
		// 	log.Info(r.Id, "Terminating request containing blacklisted path.")
		// 	r.Status = code
		// 	blacklistedIP.Add(req.RemoteAddr)
		// 	return
		// }

		firstSeg := strings.SplitN(strings.Trim(req.URL.Path, "/"), "/", 2)[0]
		if !validFirstPathSegment.Has(firstSeg) {
			code := http.StatusBadRequest
			log.HttpStatus(r.Id, w, code)
			log.Info(r.Id, "Terminating request with invalid first path segment.")
			r.Status = code
			// blacklistedIP.Add(req.RemoteAddr)
			// blacklistedPath.Add(req.URL.Path)
			return
		}

		// Redirect about page to home page.
		if req.URL.Path == "/about" {
			code := http.StatusPermanentRedirect
			url := scheme + "://" + host + "/"
			http.Redirect(w, req, url, code)
			log.Redirect(r.Id, code)
			log.Info(r.Id, "Redirecting to home page.")
			r.Status = code
			return
		}

		// Redirect subdomains to naked domain.
		if req.Host != host {
			code := http.StatusMovedPermanently
			url := scheme + "://" + host + req.RequestURI
			http.Redirect(w, req, url, code)
			log.Redirect(r.Id, code)
			log.Info(r.Id, "Redirecting subdomain to naked domain.")
			r.Status = code
			return
		}

		// Only refresh if the cache is disabled and we're not redirecting.
		if c.Cache {
			return
		}
		mustPopulateCache(c, dep.Cache)
		dep.Templates.Refresh()
	}
}

func couldBeStoryDevsPath() sd.TimedSet {
	ts := sd.NewTimedSet(0)

	// modes
	ts.Add("api")
	ts.Add("admin")
	ts.Add("account")
	ts.Add("talent")
	ts.Add("forums")
	ts.Add("library")
	ts.Add("event")

	// pages
	ts.Add("")      // home
	ts.Add("home")  // used for partial requests
	ts.Add("about") // no longer exists but not nefarious

	// public modals
	ts.Add("register")
	ts.Add("login")
	ts.Add("forgot")
	ts.Add("reserve")
	ts.Add("reserve_handle") // legacy
	ts.Add("mailing")

	// account modals
	ts.Add("delete")
	ts.Add("delete_account")
	ts.Add("email")
	ts.Add("password")
	ts.Add("persona")

	// static
	ts.Add("user")
	ts.Add("gfx")
	ts.Add("fonts")
	ts.Add("css")
	ts.Add("js")
	ts.Add("svg")
	ts.Add("robots.txt")
	ts.Add("favicon.ico")

	// actions
	ts.Add("switch")
	ts.Add("logout")

	return ts
}
