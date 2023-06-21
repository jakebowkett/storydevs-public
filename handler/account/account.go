package account

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	sd "github.com/jakebowkett/storydevs"
)

var errNoAccount = errors.New("no account attached to request")

func CreateSettings(w http.ResponseWriter, r *sd.Request) {}
func DeleteSettings(w http.ResponseWriter, r *sd.Request) {}

func Add(dep *sd.Dependencies) sd.Handler {

	as := dep.Accounts

	return func(w http.ResponseWriter, r *sd.Request) {

		cookie, err := r.Request.Cookie("token")
		if err != nil {
			return
		}

		account, err := as.RetrieveByToken(r.Id, cookie.Value, sd.AccOptRetrieve{
			Confirmed: true,
		})
		if errors.Is(err, sql.ErrNoRows) {
			return
		}
		if err != nil {
			r.Error = err
			r.Status = http.StatusBadRequest
			return
		}

		r.User = *account
	}
}

func HasNone(r *sd.Request) bool {
	_, ok := r.User.(sd.Account)
	return !ok
}

/*
IsAdmin is a skip Guard for the admin route grouping. It
returns true if there is no account or if the account does
not have admin privileges. By returning true it causes the
admin grouping to be skipped as though it doesn't exist.
*/
func NotAdmin(r *sd.Request) bool {
	account, ok := r.User.(sd.Account)
	if !ok {
		return true
	}
	p := account.ActivePersona()
	return !p.Admin.Bool
}

func Email(w http.ResponseWriter, r *sd.Request) {
	account, ok := r.User.(sd.Account)
	if !ok {
		w.WriteHeader(http.StatusNotFound)
	}
	w.Write([]byte(account.Email))
}

func Switch(dep *sd.Dependencies) sd.Handler {

	log := dep.Logger
	as := dep.Accounts

	return func(w http.ResponseWriter, r *sd.Request) {

		p := r.Vars["persona"]

		cookie, err := r.Request.Cookie("token")
		if err != nil {
			err = fmt.Errorf("cannot switch persona without token: %w", err)
			log.BadRequest(r.Id, w, err.Error())
			return
		}

		if err := as.Switch(r.Id, cookie.Value, p); err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}
	}
}

func DeletePersona(dep *sd.Dependencies) sd.Handler {

	log := dep.Logger
	as := dep.Accounts

	return func(w http.ResponseWriter, r *sd.Request) {

		p := r.Vars["persona"]
		acc := r.User.(sd.Account)

		cookie, err := r.Request.Cookie("token")
		if err != nil {
			err = fmt.Errorf("cannot delete persona without token: %w", err)
			log.BadRequest(r.Id, w, err.Error())
			return
		}

		newDefault, err := as.DeletePersona(r.Id, cookie.Value, p, acc.Id)
		if err != nil {
			log.BadRequest(r.Id, w, err.Error())
			return
		}

		w.Write([]byte(newDefault))
	}
}

func Logout(dep *sd.Dependencies) sd.Handler {

	log := dep.Logger
	as := dep.Accounts

	return func(w http.ResponseWriter, r *sd.Request) {

		// If the cookie isn't found we return a 200
		// response. The whole point of a request to
		// "/logout" is to delete the cookie anyway.
		cookie, err := r.Request.Cookie("token")
		if err != nil {
			w.WriteHeader(200)
			return
		}

		ok, err := as.Logout(r.Id, cookie.Value)
		if err != nil {
			log.BadRequest(r.Id, w, "Error while attempting to log out.")
			return
		}
		if !ok {
			log.Info(r.Id, "Failed log-out attempt.").
				Data(sd.LK_UserToken, cookie.Value)
			log.BadRequest(r.Id, w, "unable to log out")
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:   "token",
			MaxAge: -1,
		})
	}
}
