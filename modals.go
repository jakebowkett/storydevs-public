package storydevs

import "net/http"

type ForgotPassword struct {
	ResourceBase
	Identity string
	New      string
	Confirm  string
}

func (fp ForgotPassword) GetVisibility() string {
	return VisibilityPrivate
}
func (fp ForgotPassword) GetName() string {
	return ""
}

type NewPersona struct {
	ResourceBase
	Handle string
}

func (np NewPersona) GetVisibility() string {
	return VisibilityPrivate
}
func (np NewPersona) GetName() string {
	return ""
}

type Registration struct {
	ResourceBase
	Handle   string
	Email    string
	Password string
	Pronouns []string
}

func (r Registration) GetVisibility() string {
	return VisibilityPrivate
}
func (r Registration) GetName() string {
	return ""
}

type Reservation struct {
	ResourceBase
	Handle string
	Email  string
}

func (r Reservation) GetVisibility() string {
	return VisibilityPrivate
}
func (r Reservation) GetName() string {
	return ""
}

type Subscription struct {
	ResourceBase
	Email string
}

func (s Subscription) GetVisibility() string {
	return VisibilityPrivate
}
func (s Subscription) GetName() string {
	return ""
}

type Login struct {
	ResourceBase
	Identity string
	Password string
}

func (l Login) GetVisibility() string {
	return VisibilityPrivate
}
func (l Login) GetName() string {
	return ""
}

type Modals interface {
	DeleteAccount(w http.ResponseWriter, r *Request)
	Persona(w http.ResponseWriter, r *Request)
	Mailing(w http.ResponseWriter, r *Request)
	Reserve(w http.ResponseWriter, r *Request)
	Register(w http.ResponseWriter, r *Request)
	ForgotPassword(w http.ResponseWriter, r *Request)
	Login(w http.ResponseWriter, r *Request)
	Email(w http.ResponseWriter, r *Request)
	Password(w http.ResponseWriter, r *Request)
	ConfirmFull(w http.ResponseWriter, r *Request)
	ConfirmPartial(w http.ResponseWriter, r *Request)
}
