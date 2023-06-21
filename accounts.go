package storydevs

import (
	crypto "crypto/rand"
	"errors"

	"github.com/eknkc/basex"
)

type ChangeEmail struct {
	ResourceBase
	Password string
	New      string
	Confirm  string
}

func (c ChangeEmail) GetVisibility() string {
	return VisibilityPrivate
}
func (c ChangeEmail) GetName() string {
	return "Change Email"
}

type ChangePassword struct {
	ResourceBase
	Password string
	New      string
	Confirm  string
}

func (c ChangePassword) GetVisibility() string {
	return VisibilityPrivate
}
func (c ChangePassword) GetName() string {
	return "Change Password"
}

type SettingsIdentity struct {
	ResourceBase
	Avatar   File
	Handle   string
	Name     string
	Pronouns []string
}

func (si SettingsIdentity) GetVisibility() string {
	return VisibilityPrivate
}
func (si SettingsIdentity) GetName() string {
	return "Identity"
}

type SettingsPrivacy struct {
	ResourceBase
	Visibility string
}

func (sp SettingsPrivacy) GetVisibility() string {
	return VisibilityPrivate
}
func (sp SettingsPrivacy) GetName() string {
	return "Privacy"
}

type Settings struct {
	ResourceBase
	Field
}

func (s Settings) GetVisibility() string {
	return VisibilityPrivate
}
func (s Settings) GetName() string {
	return s.Name
}

type SettingAuth struct {
	NewPass string
	Pass    string
}

const base62 = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

/*
Base62 is used to generate confirmation
codes and authentication tokens.
*/
func Base62(bytes int) (string, error) {

	enc, err := basex.NewEncoding(base62)
	if err != nil {
		return "", err
	}

	p := make([]byte, bytes)
	_, err = crypto.Read(p)
	if err != nil {
		return "", err
	}

	return enc.Encode(p), nil
}

var ErrInvalidLoginAttempt = errors.New("invalid login attempt")

type Password interface {
	Hash(pass string) (hash string, err error)
	Compare(pass, hash string) (ok bool, err error)
	SetCost(cost int)
}

type AuthedUser struct {
	Token   string
	Account *Account
}

type Account struct {
	Id int64

	Email string
	Pass  string

	Code NullString

	Created int64

	Personas []Persona
}

// Mod privileges.
type Privileges struct {
	Approve    NullBool // can approve resources
	Lock       NullBool // can prevent users from making changes/additions to resources
	Edit       NullBool // can make edits to resources
	Hide       NullBool // can hide resources - prefer over delete
	Destroy    NullBool // can delete resources - reserve for illegal content & spam
	Mute       NullBool // can mute users
	Ban        NullBool // can ban users
	Tags       NullBool // can manage resource tags (i.e., merge, re-name, etc)
	Categories NullBool // can add/remove resource categories (effectively "move")
	Logs       NullBool // can view moderator action logs
	GivePriv   NullBool // can give privileges to users
	TakePriv   NullBool // can take away privileges from users
}

type Persona struct {
	AccId int64
	Id    int64

	Name       NullString
	Avatar     FileName
	Handle     string
	Slug       string
	Visibility string
	Pronouns   []string

	Created int

	Admin   NullBool
	Deleted NullBool
	Active  bool
	Default bool `db:"default_p"`

	// Privileges
}

func (a Account) Handles() (handles []string) {
	for _, p := range a.Personas {
		handles = append(handles, p.Handle)
	}
	return handles
}

func (a Account) ActivePersona() Persona {
	if len(a.Personas) == 0 {
		return Persona{}
	}
	idx := 0
	for i, p := range a.Personas {
		if p.Active {
			return p
		}
		if p.Default {
			idx = i
		}
	}
	return a.Personas[idx]
}

const (
	InviteRequired = iota
	InviteOptional
)
const (
	ConfirmManual = iota
	ConfirmAuto
)

type AccOptRetrieve struct {
	Password       bool
	Confirmed      bool
	DeletedAccount bool
	DeletedPersona bool
}

type Accounts interface {
	Mailing(reqId, email string) (Feedback, error)
	ConfirmSubscription(reqId, code string) (bool, error)

	Reserve(reqId, handle, email string) (Feedback, error)
	ConfirmReservation(reqId, code string) (bool, error)

	ChangeEmail(reqId string, c *ChangeEmail, accId int64) (Feedback, error)
	ConfirmEmail(reqId, code string) (bool, error)

	ForgotPassword(reqId string, forgot *ForgotPassword) (Feedback, error)

	ChangePassword(reqId string, c *ChangePassword, accId int64) (Feedback, error)
	ConfirmPassword(reqId, code string) (bool, error)

	NewPersona(reqId string, accId int64, np *NewPersona) (Feedback, error)
	DeletePersona(reqId, token, slug string, accId int64) (string, error)

	Create(reqId string, r *Registration, confirm int, isAdmin bool) (Feedback, error)
	ConfirmAccount(reqId, code string) (bool, error)
	Delete(reqId string, id int64) error

	RetrieveByToken(reqId, token string, o AccOptRetrieve) (*Account, error)
	RetrieveById(reqId string, id int64, o AccOptRetrieve) (*Account, error)
	RetrieveByEmail(reqId, email string, o AccOptRetrieve) (*Account, error)
	RetrieveByHandle(reqId, handle string, o AccOptRetrieve) (*Account, error)

	Switch(reqId, token, slug string) (err error)
	Login(reqId, identity, pass string) (auth *AuthedUser, err error)
	Logout(reqId, token string) (ok bool, err error)
}
