package modal

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	sd "github.com/jakebowkett/storydevs"
	"github.com/jakebowkett/storydevs/handler"
	"github.com/jakebowkett/storydevs/handler/form"
)

type Service struct {
	*sd.Dependencies
}

type rw = http.ResponseWriter

func (s *Service) DeleteAccount(w http.ResponseWriter, r *sd.Request) {
	accId := r.User.(sd.Account).Id
	if err := s.Accounts.Delete(r.Id, accId); err != nil {
		s.Logger.BadRequest(r.Id, w, err.Error())
		return
	}
}

func (s *Service) Persona(w http.ResponseWriter, r *sd.Request) {

	body := &sd.NewPersona{}
	name := "persona"

	if err := s.extractAndValidate(w, r, name, body); err != nil {
		s.Logger.BadRequest(r.Id, w, err.Error())
		return
	}
	fb, err := s.Accounts.NewPersona(
		r.Id,
		r.User.(sd.Account).Id,
		body,
	)
	if err != nil {
		s.Logger.BadRequest(r.Id, w, err.Error())
		return
	}

	m := make(map[string]interface{})

	if len(fb) > 0 {
		m["feedback"] = fb
		s.jsonResponse(w, r, m)
		return
	}

	m["slug"] = body.Slug
	m["handle"] = body.Handle
	s.jsonResponse(w, r, m)
}

func (s *Service) Register(w http.ResponseWriter, r *sd.Request) {

	body := &sd.Registration{}
	name := "register"

	if err := s.extractAndValidate(w, r, name, body); err != nil {
		s.Logger.BadRequest(r.Id, w, err.Error())
		return
	}
	fb, err := s.Accounts.Create(
		r.Id,
		body,
		sd.ConfirmManual,
		false,
	)
	if err != nil {
		s.Logger.BadRequest(r.Id, w, err.Error())
		return
	}

	s.modalResponse(w, r, "register_confirm", fb, body.Email)
}

func (s *Service) Email(w http.ResponseWriter, r *sd.Request) {

	body := &sd.ChangeEmail{}
	name := "email"

	if err := s.extractAndValidate(w, r, name, body); err != nil {
		s.Logger.BadRequest(r.Id, w, err.Error())
		return
	}
	accId := (r.User.(sd.Account)).Id
	fb, err := s.Accounts.ChangeEmail(r.Id, body, accId)
	if err != nil {
		s.Logger.BadRequest(r.Id, w, err.Error())
		return
	}

	s.modalResponse(w, r, "email_confirm", fb, body.New)
}

/*
ForgotPassword uses log.Info and generally does not give any
feedback to the user to prevent them from knowing whether
the account does or does not exist and to prevent them from
knowing if the email was sent. Might want to do this for
Login at some point too.
*/
func (s *Service) ForgotPassword(w http.ResponseWriter, r *sd.Request) {

	body := &sd.ForgotPassword{}
	name := "forgot"
	log := s.Logger

	if err := s.extractAndValidate(w, r, name, body); err != nil {
		log.BadRequest(r.Id, w, err.Error())
		return
	}
	fb, err := s.Accounts.ForgotPassword(r.Id, body)
	if err != nil {
		log.BadRequest(r.Id, w, err.Error())
		return
	}

	s.modalResponse(w, r, "forgot_confirm", fb, "your address.")
}

func (s *Service) Password(w http.ResponseWriter, r *sd.Request) {

	body := &sd.ChangePassword{}
	name := "password"
	acc := r.User.(sd.Account)

	if err := s.extractAndValidate(w, r, name, body); err != nil {
		s.Logger.BadRequest(r.Id, w, err.Error())
		return
	}
	fb, err := s.Accounts.ChangePassword(r.Id, body, acc.Id)
	if err != nil {
		s.Logger.BadRequest(r.Id, w, err.Error())
		return
	}

	s.modalResponse(w, r, "password_confirm", fb, acc.Email)
}

func (s *Service) Mailing(w http.ResponseWriter, r *sd.Request) {

	body := &sd.Subscription{}
	name := "mailing"

	if err := s.extractAndValidate(w, r, name, body); err != nil {
		s.Logger.BadRequest(r.Id, w, err.Error())
		return
	}
	fb, err := s.Accounts.Mailing(r.Id, body.Email)
	if err != nil {
		s.Logger.BadRequest(r.Id, w, err.Error())
		return
	}

	s.modalResponse(w, r, "mailing_confirm", fb, body.Email)
}

func (s *Service) Reserve(w http.ResponseWriter, r *sd.Request) {

	body := &sd.Reservation{}
	name := "reserve"

	if err := s.extractAndValidate(w, r, name, body); err != nil {
		s.Logger.BadRequest(r.Id, w, err.Error())
		return
	}
	fb, err := s.Accounts.Reserve(r.Id, body.Handle, body.Email)
	if err != nil {
		s.Logger.BadRequest(r.Id, w, err.Error())
		return
	}

	s.modalResponse(w, r, "reserve_confirm", fb, body.Email)
}

func (s *Service) Login(w http.ResponseWriter, r *sd.Request) {

	if r.User != nil {
		s.Logger.BadRequest(r.Id, w, "Client already logged in.")
		return
	}

	body := &sd.Login{}
	name := "login"

	if err := s.extractAndValidate(w, r, name, body); err != nil {
		s.Logger.BadRequest(r.Id, w, err.Error())
		return
	}

	authedUser, err := s.Accounts.Login(r.Id, body.Identity, body.Password)
	if errors.Is(err, sd.ErrInvalidLoginAttempt) {
		fb := make(sd.Feedback)
		fb.Add("general", "Unable to log in. Please check your details.")
		s.jsonResponse(w, r, map[string]interface{}{"feedback": fb})
		return
	}
	if err != nil {
		s.Logger.BadRequest(r.Id, w, err.Error())
		return
	}

	var secure bool
	var sameSite http.SameSite
	if !s.Config.Dev {
		secure = true
		sameSite = http.SameSiteStrictMode
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "token",
		Value:    authedUser.Token,
		HttpOnly: true,
		SameSite: sameSite,
		Secure:   secure,
		MaxAge:   60 * 60 * 24 * 30 * 6, // roughly 6 months
		Path:     "/",
	})

	m := make(map[string]interface{})

	ps, err := s.Templates.Render("persona_switcher", struct {
		Account *sd.Account
		View    string
	}{
		Account: authedUser.Account,
	})
	if err != nil {
		s.Logger.BadRequest(r.Id, w, err.Error())
		return
	}
	m["ps"] = string(ps)

	p := authedUser.Account.ActivePersona()
	if p.Admin.Bool {

		link, err := s.Templates.Render("admin_link", "")
		if err != nil {
			s.Logger.BadRequest(r.Id, w, err.Error())
			return
		}
		m["link"] = string(link)

		var mm []map[string]interface{}
		for _, md := range s.ViewData.Mode {
			if !md.AdminOnly {
				continue
			}
			meta := make(map[string]interface{})
			meta["name"] = md.Name
			meta["title"] = md.Title
			meta["resourceName"] = md.ResourceName
			meta["resourcePlural"] = md.ResourceName
			meta["resourceColumn"] = md.ResourceName
			meta["logoutRemove"] = md.LogoutRemove
			mm = append(mm, meta)
		}
		if len(mm) > 0 {
			m["meta"] = mm
		}
	}

	s.jsonResponse(w, r, m)
}

func (s *Service) Logout(w http.ResponseWriter, r *sd.Request) {

	// If the cookie isn't found we return a 200
	// response. The whole point of a request to
	// "/logout" is to delete the cookie anyway.
	cookie, err := r.Request.Cookie("token")
	if err != nil {
		w.WriteHeader(200)
		return
	}

	ok, err := s.Accounts.Logout(r.Id, cookie.Value)
	if err != nil {
		s.Logger.BadRequest(r.Id, w, "Error while attempting to log out.")
		return
	}
	if !ok {
		s.Logger.Info(r.Id, "Failed log-out attempt.").
			Data(sd.LK_UserToken, cookie.Value)
		s.Logger.BadRequest(r.Id, w, "unable to log out")
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "token",
		MaxAge: -1,
	})
}

func (s *Service) ConfirmFull(w http.ResponseWriter, r *sd.Request) {

	kind := r.Vars["kind"]
	if kind == "reserve_handle" {
		kind = "reserve"
	}
	code := r.Vars["code"]
	ok, err := s.confirmPartial(r.Id, kind, code)
	if err != nil {
		s.Logger.BadRequest(r.Id, w, err.Error())
		return
	}
	if ok {
		r.Vars["modal"] = kind + "_success"
	} else {
		r.Vars["modal"] = "confirm_fail"
	}

	Full(s.Dependencies)(w, r)
}

func (s *Service) ConfirmPartial(w http.ResponseWriter, r *sd.Request) {

	kind := r.Vars["kind"]
	body := &struct{ Code string }{}
	err := handler.ExtractJson(w, r.Request, body, s.Config.MaxForm[kind])
	if err != nil {
		s.Logger.BadRequest(r.Id, w, err.Error())
		return
	}
	code := body.Code

	ok, err := s.confirmPartial(r.Id, kind, code)
	if err != nil {
		s.Logger.BadRequest(r.Id, w, err.Error())
		return
	}

	var name string
	fb := make(sd.Feedback)
	if ok {
		name = kind + "_success"
	} else {
		name = "confirm_fail"
		fb.Add("code", "Invalid confirmation code.")
	}

	s.modalResponse(w, r, name, fb)
}

func (s *Service) confirmPartial(reqId, kind, code string) (bool, error) {

	as := s.Accounts

	var errKind string
	var err error
	var ok bool

	switch kind {
	case "register":
		ok, err = as.ConfirmAccount(reqId, code)
		errKind = "account"
	case "reserve":
		ok, err = as.ConfirmReservation(reqId, code)
		errKind = "reservation"
	case "mailing":
		ok, err = as.ConfirmSubscription(reqId, code)
		errKind = "subscription"
	case "email":
		ok, err = as.ConfirmEmail(reqId, code)
		errKind = "email change"
	case "password":
		ok, err = as.ConfirmPassword(reqId, code)
		errKind = "password change"
	}
	if err != nil {
		return false, fmt.Errorf("Error while attempting to confirm %s.", errKind)
	}
	return ok, nil
}

func (s *Service) modalResponse(
	w rw,
	r *sd.Request,
	modal string,
	fb sd.Feedback,
	data ...interface{},
) {

	m := make(map[string]interface{})

	if len(fb) > 0 {
		m["feedback"] = fb
		s.jsonResponse(w, r, m)
		return
	}

	md := s.ViewData.Modal[modal]

	/*
	   We copy fields here because we're rendering a
	   different user's email address into the modal
	   each time.
	*/
	if strings.HasSuffix(md.Name, "confirm") {
		var ff sd.Fields
		for _, f := range md.Field {
			f.Context = fmt.Sprintf(f.Context, data...)
			ff = append(ff, f)
		}
		md.Field = ff
	}

	v, err := s.Templates.Render("modal.html", md)
	if err != nil {
		s.Logger.BadRequest(r.Id, w, err.Error())
		return
	}
	m["modal"] = string(v)
	s.jsonResponse(w, r, m)
}

func (s *Service) jsonResponse(w rw, r *sd.Request, data interface{}) {

	d, err := json.Marshal(data)
	if err != nil {
		s.Logger.BadRequest(r.Id, w, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	handler.Gzip(w, r, d, 200, s.Logger)
}

func (s *Service) extractAndValidate(
	w rw,
	r *sd.Request,
	name string,
	res sd.Resource,
) error {
	err := handler.ExtractJson(w, r.Request, res, s.Config.MaxForm[name])
	if err != nil {
		return err
	}
	mapping, ok := s.ResourceMapping[name]
	if !ok {
		return fmt.Errorf("No mapping for %s.", name)
	}
	modalData, ok := s.ViewData.Modal[name]
	if !ok {
		return fmt.Errorf("No modal fields for %s.", name)
	}
	_, err = form.Validate(r.Id, s.Config, s.Logger, name, res, mapping, modalData.Field, false, s.TryerDisk)
	return err
}
