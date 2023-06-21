package service

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jakebowkett/go-gen/gen"
	sd "github.com/jakebowkett/storydevs"
)

type Account struct {
	*sd.Dependencies
}

func newCode() (string, error) {
	return sd.Base62(64)
}

func newToken() (string, error) {
	return sd.Base62(128)
}

// ILIKE is used for case-insensitive handle and email comparisions.
func (as *Account) handleAndEmailFree(
	tx sd.Tx,
	fb sd.Feedback,
	handle string,
	email string,
	registering bool,
) (
	err error,
) {

	var exists bool

	/*
	   If we're registering an account we allow redeeming a
	   reserved handle if the exact handle and email is used.
	*/
	if registering {
		exists, err := tx.Exists(`
			FROM
				reserved
			WHERE
				handle ILIKE $1 AND
				email ILIKE $2 AND
				code IS NULL`,
			handle, email)
		if err != nil {
			return err
		}
		if exists {
			goto redeemHandle
		}
	}

	/*
		We check whether redeemed is null because the account/persona
		checks below will catch if the handle is in use by them. We're
		basically trying to allow for using a *redeemed* handle that
		is no longer in use by a account/persona.

		For example, someone reserves "first" and then redeems it. Then
		they change their handle to "second". In this case the handle
		"first" is now free and should be allowed to be used.
	*/
	exists, err = tx.Exists(`
		FROM
			reserved
		WHERE
			reserved.handle ILIKE $1 AND
			reserved.code IS NULL AND
			redeemed IS NULL`,
		handle)
	if err != nil {
		return err
	}
	if exists {
		fb.Add("handle", "handle is already in use")
		return nil
	}

	exists, err = tx.Exists(`
		FROM
			reserved
		WHERE
			reserved.email ILIKE $1 AND
			reserved.code IS NULL`,
		email)
	if err != nil {
		return err
	}
	if exists {
		fb.Add("email", "email is already in use")
		return nil
	}

redeemHandle:

	exists, err = tx.Exists(`
		FROM
			accounts,
			personas
		WHERE
			accounts.id = personas.acc_id AND
			personas.handle ILIKE $1 AND
			accounts.deleted IS NULL AND
			accounts.code IS NULL`,
		handle)
	if err != nil {
		return err
	}
	if exists {
		fb.Add("handle", "handle is already in use")
		return nil
	}

	exists, err = tx.Exists(`
		FROM
			accounts
		WHERE
			email ILIKE $1 AND
			deleted IS NULL AND
			code IS NULL`,
		email)
	if err != nil {
		return err
	}
	if exists {
		fb.Add("email", "email is already in use")
		return nil
	}

	return nil
}

func (as *Account) NewPersona(
	reqId string,
	accId int64,
	np *sd.NewPersona,
) (sd.Feedback, error) {

	log := as.Logger
	fb := make(sd.Feedback)

	if np == nil {
		return nil, errors.New("supplied new persona is nil")
	}

	var pId int64
	var pSlug string

	errs, err := as.TryerTx.Try(func() error {

		tx, err := as.Db.Begin()
		if err != nil {
			return err
		}

		var count int
		err = tx.Get(&count, `
			SELECT
				COUNT(*)
			FROM
				accounts,
				personas
			WHERE
				accounts.id = $1 AND
				accounts.id = personas.acc_id AND
				personas.deleted IS NULL`,
			accId)
		if err != nil {
			return tx.Rollback(err)
		}
		if count >= as.Config.MaxPersonas {
			msg := "account already has maximum allowed personas"
			return tx.Rollback(errors.New(msg))
		}

		pSlug, _ = gen.AlphaNum(11)
		now := time.Now().Unix()

		exists, err := tx.Exists(`
			FROM
				reserved
			WHERE
				reserved.handle ILIKE $1 AND
				reserved.code IS NULL`,
			np.Handle)
		if err != nil {
			return tx.Rollback(err)
		}
		if exists {
			fb.Add("handle", "handle is already in use")
			return tx.Rollback(nil)
		}

		exists, err = tx.Exists(`
			FROM
				accounts,
				personas
			WHERE
				accounts.id = personas.acc_id AND
				personas.handle ILIKE $1 AND
				accounts.code IS NULL AND
				personas.deleted IS NULL`,
			np.Handle)
		if err != nil {
			return tx.Rollback(err)
		}
		if exists {
			fb.Add("handle", "handle is already in use")
			return tx.Rollback(nil)
		}

		err = tx.Get(&pId, `
			INSERT INTO personas (
				acc_id,
				slug,
				handle,
				name,
				default_p,
				created,
				updated,
				visibility
			)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING
				id`,
			accId,
			pSlug,
			np.Handle,
			np.Handle,
			false,
			now,
			now,
			sd.VisibilityPublic,
		)
		if err != nil {
			return tx.Rollback(err)
		}

		return tx.Commit()
	})

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_AccId, accId).
			Data(sd.LK_PersHandle, np.Handle)
		return nil, errors.New("Unable to create new persona.")
	}

	if len(fb) > 0 {
		return fb, nil
	}

	// We need this to give to the client-side.
	np.Slug = pSlug

	log.Info(reqId, "Created new persona.").
		Data(sd.LK_AccId, accId).
		Data(sd.LK_PersId, pId).
		Data(sd.LK_PersHandle, np.Handle)

	return fb, nil
}

func (as *Account) Create(
	reqId string,
	r *sd.Registration,
	confirmManual int,
	isAdmin bool,
) (sd.Feedback, error) {

	log := as.Logger
	fb := make(sd.Feedback)

	if r == nil {
		return nil, errors.New("supplied account is nil")
	}

	handle := r.Handle
	email := r.Email
	pass := r.Password
	pronouns := r.Pronouns

	if len(fb) > 0 {
		return fb, nil
	}

	var admin *bool
	var code *string
	var hash string
	var accId int64
	var pId int64
	var pSlug string

	if isAdmin {
		admin = func(b bool) *bool { return &b }(true)
	}

	errs, err := as.TryerTx.Try(func() error {

		tx, err := as.Db.Begin()
		if err != nil {
			return err
		}

		err = as.handleAndEmailFree(tx, fb, handle, email, true)
		if err != nil {
			return tx.Rollback(err)
		}
		if len(fb) > 0 {
			return tx.Rollback(nil)
		}

		now := time.Now().Unix()
		if code == nil && confirmManual == sd.ConfirmManual {
			c, err := newCode()
			if err != nil {
				return tx.Rollback(err)
			}
			code = &c
		}

		/*
			We hash pass at the last moment so that no time
			is wasted if something went wrong earlier. First
			we check to see if pass has already been hashed
			to avoid doing it again on a retry.
		*/
		if hash == "" {
			hash, err = as.Password.Hash(pass)
			if err != nil {
				return tx.Rollback(err)
			}
		}

		err = tx.Get(&accId, `
			INSERT INTO accounts (
				email,
				pass,
				created,
				updated,
				code
			)
			VALUES
				($1, $2, $3, $4, $5)
			RETURNING
				id`,
			email,
			hash,
			now,
			now,
			code,
		)
		if err != nil {
			return tx.Rollback(err)
		}

		pSlug, _ = gen.AlphaNum(11)

		err = tx.Get(&pId, `
			INSERT INTO personas (
				acc_id,
				slug,
				handle,
				name,
				default_p,
				created,
				updated,
				visibility,
				admin
			)
			VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9)
			RETURNING
				id`,
			accId,
			pSlug,
			handle,
			handle,
			true,
			now,
			now,
			sd.VisibilityPublic,
			admin,
		)
		if err != nil {
			return tx.Rollback(err)
		}

		for _, pn := range pronouns {
			q := `INSERT INTO pronouns (ref_id, pronoun) VALUES ($1, $2)`
			if _, err = tx.Exec(q, pId, pn); err != nil {
				return tx.Rollback(err)
			}
		}

		return tx.Commit()
	})

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_AccEmail, email).
			Data(sd.LK_PersHandle, handle)
		if lastErrSqlNoRows(errs) {
			return fb, nil
		}
		return nil, errors.New("Unable to complete account registration transaction.")
	}

	if len(fb) > 0 {
		return fb, nil
	}

	if confirmManual == sd.ConfirmManual {
		errs, err = as.TryerEmail.Try(func() error {
			subject := "Confirm your StoryDevs account"
			c := ""
			if code != nil {
				c = *code
			}
			return as.sendConfirmationEmail("register", email, subject, c)
		})
		if err != nil {
			log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
				Data(sd.LK_RetryAttemptsEmail, len(errs)).
				Data(sd.LK_AccId, accId).
				Data(sd.LK_AccEmail, email).
				Data(sd.LK_PersId, pId).
				Data(sd.LK_PersSlug, pSlug).
				Data(sd.LK_PersHandle, handle)
			if err := as.Delete(reqId, accId); err != nil {
				return nil, err
			}
			return nil, errors.New("Unable to send account confirmation email.")
		}
	}

	logEntry := log.Info(reqId, "Created account.").
		Data(sd.LK_AccId, accId).
		Data(sd.LK_AccEmail, email).
		Data(sd.LK_PersId, pId).
		Data(sd.LK_PersSlug, pSlug).
		Data(sd.LK_PersHandle, handle)
	if code != nil {
		logEntry.Data(sd.LK_AccCode, *code)
	}

	return fb, nil
}

func (as *Account) ForgotPassword(
	reqId string,
	forgot *sd.ForgotPassword,
) (sd.Feedback, error) {

	identity := forgot.Identity
	acc := &sd.Account{}
	log := as.Logger
	fb := make(sd.Feedback)

	if forgot.New != forgot.Confirm {
		fb.Add("confirm", "Passwords do not match.")
		return fb, nil
	}

	code, err := newCode()
	if err != nil {
		return nil, err
	}

	hash, err := as.Password.Hash(forgot.New)
	if err != nil {
		return nil, err
	}

	errs, err := as.TryerTx.Try(func() error {

		tx, err := as.Db.Begin()
		if err != nil {
			return err
		}

		if strings.Contains(identity, "@") {
			err = tx.Get(acc, `
				SELECT
					id,
					email
				FROM
					accounts
				WHERE
					email = $1 AND
					code IS NULL AND
					deleted IS NULL`,
				identity,
			)
		} else {
			err = tx.Get(acc, `
				SELECT
					accounts.id,
					accounts.email
				FROM
					accounts,
					personas
				WHERE
					personas.handle = $1 AND
					personas.deleted IS NULL AND
					personas.acc_id = accounts.id AND
					accounts.code IS NULL AND
					accounts.deleted IS NULL
				`,
				identity)
		}
		if err != nil {
			return tx.Rollback(err)
		}

		_, err = tx.Exec(`
			INSERT INTO change_password (
				ref_id,
				pass,
				code
			)
			VALUES
				($1, $2, $3)`,
			acc.Id,
			hash,
			code,
		)
		if err != nil {
			return tx.Rollback(err)
		}

		return tx.Commit()
	})
	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_AccEmail, acc.Email)
		if lastErrSqlNoRows(errs) {
			return nil, nil
		}
		return nil, errors.New("Unable to complete forgot password transaction.")
	}

	errs, err = as.TryerEmail.Try(func() error {
		subject := "Forgotten Password on StoryDevs"
		return as.sendConfirmationEmail("forgot", acc.Email, subject, code)
	})
	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsEmail, len(errs)).
			Data(sd.LK_AccId, acc.Id).
			Data(sd.LK_AccEmail, acc.Email)
		return nil, errors.New("Unable to send forgot password email.")
	}

	log.Info(reqId, "Sent forgotten password email.").
		Data(sd.LK_AccId, acc.Id).
		Data(sd.LK_AccEmail, acc.Email).
		Data(sd.LK_AccForgotCode, code)

	return nil, nil
}

func (as *Account) ChangeEmail(
	reqId string,
	change *sd.ChangeEmail,
	accId int64,
) (sd.Feedback, error) {

	log := as.Logger
	db := as.Db
	fb := make(sd.Feedback)

	if change.New != change.Confirm {
		fb.Add("confirm", "Email addresses do not match.")
		return fb, nil
	}

	code, err := newCode()
	if err != nil {
		return nil, err
	}

	current := struct {
		Pass  string
		Email string
	}{}

	errs, err := as.TryerTx.Try(func() error {

		tx, err := db.Begin()
		if err != nil {
			return err
		}

		// Check email address is not already in use.
		exists, err := tx.Exists(`
			FROM
				accounts
			WHERE
				email ILIKE $1 AND
				code IS NULL`,
			change.New)
		if err != nil {
			return tx.Rollback(err)
		}
		if exists {
			fb.Add("new", "This address is already in use.")
			return tx.Rollback(errors.New("service: new email already exists"))
		}

		/*
			Compare password. We hash here after checking
			for availability to avoid an expensive operation
			(hashing) when it's not needed.
		*/
		err = tx.Get(&current, `
			SELECT
				pass,
				email
			FROM
				accounts
			WHERE
				id = $1`,
			accId)
		if err != nil {
			return tx.Rollback(err)
		}
		ok, err := as.Password.Compare(change.Password, current.Pass)
		if err != nil {
			return tx.Rollback(err)
		}
		if !ok {
			fb.Add("password", "Incorrect password.")
			return tx.Rollback(errors.New("incorrect password"))
		}

		/*
			Insert new email into change_email table.
			We preserve the old email until the new
			address confirms it.
		*/
		_, err = tx.Exec(`
			INSERT INTO change_email (
				ref_id,
				email,
				code
			)
			VALUES
				($1, $2, $3)`,
			accId,
			change.New,
			code,
		)
		if err != nil {
			return tx.Rollback(err)
		}

		return tx.Commit()
	})

	if len(fb) > 0 {
		return fb, nil
	}

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_AccId, accId)
		return nil, errors.New("Unable to add email change entry.")
	}

	subject := "Confirm your new address"
	err = as.sendConfirmationEmail("email", change.New, subject, code)
	if err != nil {
		log.Error(reqId, err.Error()).
			Data(sd.LK_AccId, accId).
			Data(sd.LK_AccEmail, current.Email).
			Data(sd.LK_AccEmailNew, change.New)
		return nil, errors.New("Unable to send change email confirmation email.")
	}

	log.Info(reqId, "Created change email entry.").
		Data(sd.LK_AccId, accId).
		Data(sd.LK_AccEmail, current.Email).
		Data(sd.LK_AccEmailNew, change.New).
		Data(sd.LK_AccEmailCode, code)

	return nil, nil
}

func (as *Account) ConfirmEmail(reqId, code string) (bool, error) {

	data := struct {
		Id       int64
		OldEmail string
		NewEmail string
	}{}

	errs, err := as.TryerTx.Try(func() error {

		tx, err := as.Db.Begin()
		if err != nil {
			return err
		}

		err = tx.Get(&data, `
			SELECT
				accounts.id,
				accounts.email AS oldemail,
				change_email.email AS newemail
			FROM
				accounts,
				change_email
			WHERE
				accounts.id = change_email.ref_id AND
				change_email.code = $1
			`, code)
		if err != nil {
			return tx.Rollback(err)
		}

		_, err = tx.Exec(`
			UPDATE
				accounts
			SET
				email = change_email.email
			FROM
				change_email
			WHERE
				accounts.id = change_email.ref_id AND
				change_email.code = $1
			`, code)
		if err != nil {
			return tx.Rollback(err)
		}

		_, err = tx.Exec(`
			DELETE FROM
				change_email
			WHERE
				ref_id = $1
			`, data.Id)
		if err != nil {
			return tx.Rollback(err)
		}

		return tx.Commit()
	})

	log := as.Logger

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_AccEmailCode, code)
		return false, errors.New("Unable to confirm email change.")
	}

	log.Info(reqId, "Changed email.").
		Data(sd.LK_AccId, data.Id).
		Data(sd.LK_AccEmailOld, data.OldEmail).
		Data(sd.LK_AccEmailNew, data.NewEmail)

	return true, nil
}

func (as *Account) ChangePassword(
	reqId string,
	change *sd.ChangePassword,
	accId int64,
) (sd.Feedback, error) {

	log := as.Logger
	db := as.Db
	fb := make(sd.Feedback)

	if change.New != change.Confirm {
		fb.Add("confirm", "Passwords do not match.")
		return fb, nil
	}

	code, err := newCode()
	if err != nil {
		return nil, err
	}

	current := struct {
		Pass  string
		Email string
	}{}

	errs, err := as.TryerTx.Try(func() error {

		tx, err := db.Begin()
		if err != nil {
			return err
		}

		// Compare password.
		err = tx.Get(&current, `
			SELECT
				pass,
				email
			FROM
				accounts
			WHERE
				id = $1`,
			accId)
		if err != nil {
			return tx.Rollback(err)
		}
		ok, err := as.Password.Compare(change.Password, current.Pass)
		if err != nil {
			return tx.Rollback(err)
		}
		if !ok {
			fb.Add("password", "Incorrect password.")
			return tx.Rollback(errors.New("incorrect password"))
		}

		/*
			Hash new password and insert it into change_password
			table. Even though it's not in use yet we still hash
			it as it's sensitive information. We preserve the old
			password until it's confirmed.
		*/
		newHash, err := as.Password.Hash(change.New)
		if err != nil {
			return tx.Rollback(err)
		}
		_, err = tx.Exec(`
			INSERT INTO change_password (
				ref_id,
				pass,
				code
			)
			VALUES
				($1, $2, $3)`,
			accId,
			newHash,
			code,
		)
		if err != nil {
			return tx.Rollback(err)
		}

		return tx.Commit()
	})

	if len(fb) > 0 {
		return fb, nil
	}

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_AccId, accId)
		return nil, errors.New("Unable to add password change entry.")
	}

	subject := "Confirm your new password"
	err = as.sendConfirmationEmail("password", current.Email, subject, code)
	if err != nil {
		log.Error(reqId, err.Error()).
			Data(sd.LK_AccId, accId).
			Data(sd.LK_AccEmail, current.Email)
		return nil, errors.New("Unable to send change password confirmation email.")
	}

	log.Info(reqId, "Created change password entry.").
		Data(sd.LK_AccId, accId).
		Data(sd.LK_AccEmail, current.Email).
		Data(sd.LK_AccPassCode, code)

	return nil, nil
}

func (as *Account) ConfirmPassword(reqId, code string) (bool, error) {

	data := struct {
		Id    int64
		Email string
	}{}

	errs, err := as.TryerTx.Try(func() error {

		tx, err := as.Db.Begin()
		if err != nil {
			return err
		}

		err = tx.Get(&data, `
			UPDATE
				accounts
			SET
				pass = change_password.pass
			FROM
				change_password
			WHERE
				accounts.id = change_password.ref_id AND
				change_password.code = $1
			RETURNING
				accounts.id,
				accounts.email`,
			code)
		if err != nil {
			return tx.Rollback(err)
		}

		_, err = tx.Exec(`
			DELETE FROM
				change_password
			WHERE
				ref_id = $1`,
			data.Id)
		if err != nil {
			return tx.Rollback(err)
		}

		return tx.Commit()
	})

	log := as.Logger

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_AccEmailCode, code)
		return false, errors.New("Unable to confirm password change.")
	}

	log.Info(reqId, "Changed password.").
		Data(sd.LK_AccId, data.Id).
		Data(sd.LK_AccEmail, data.Email)

	return true, nil
}

func (as *Account) Mailing(reqId, email string) (sd.Feedback, error) {

	log := as.Logger
	fb := make(sd.Feedback)

	var code string
	var id int64

	errs, err := as.TryerTx.Try(func() error {

		tx, err := as.Db.Begin()
		if err != nil {
			return err
		}

		exists, err := tx.Exists(`
			FROM
				mailing
			WHERE
				email ILIKE $1 AND
				code IS NULL`,
			email)
		if err != nil {
			return tx.Rollback(err)
		}
		if exists {
			fb.Add("email", "Address is already subscribed to the mailing list.")
			return tx.Rollback(nil)
		}

		code, err = newCode()
		if err != nil {
			return tx.Rollback(err)
		}

		err = tx.Get(&id, `
			INSERT INTO mailing (
				email,
				code
			)
			VALUES
				($1, $2)
			RETURNING
				id`,
			email, code,
		)
		if err != nil {
			return tx.Rollback(err)
		}

		return tx.Commit()
	})

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_SubscriptionEmail, email)
		return nil, errors.New("Unable to complete mailing list transaction.")
	}
	if len(fb) > 0 {
		return fb, nil
	}

	errs, err = as.TryerEmail.Try(func() error {
		subject := "Confirm your StoryDevs mailing list subscription"
		return as.sendConfirmationEmail("mailing", email, subject, code)
	})
	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsEmail, len(errs)).
			Data(sd.LK_SubscriptionId, id).
			Data(sd.LK_SubscriptionEmail, email)
		return nil, errors.New("Unable to send mailing list confirmation email.")
	}

	log.Info(reqId, "Created subscription.").
		Data(sd.LK_SubscriptionId, id).
		Data(sd.LK_SubscriptionEmail, email).
		Data(sd.LK_SubscriptionCode, code)

	return nil, nil
}

func (as *Account) Reserve(reqId, handle, email string) (sd.Feedback, error) {

	log := as.Logger
	fb := make(sd.Feedback)

	var code string
	var id int64

	errs, err := as.TryerTx.Try(func() error {

		tx, err := as.Db.Begin()
		if err != nil {
			return err
		}

		err = as.handleAndEmailFree(tx, fb, handle, email, false)
		if err != nil {
			return tx.Rollback(err)
		}
		if len(fb) > 0 {
			return tx.Rollback(nil)
		}

		code, err = newCode()
		if err != nil {
			return tx.Rollback(err)
		}

		err = tx.Get(&id, `
			INSERT INTO reserved (
				handle,
				email,
				code
			)
			VALUES
				($1, $2, $3)
			RETURNING
				id`,
			handle,
			email,
			code,
		)
		if err != nil {
			return tx.Rollback(err)
		}

		return tx.Commit()
	})

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_ReserveEmail, email).
			Data(sd.LK_ReserveHandle, handle)
		return nil, errors.New("Unable to complete handle reservation transaction.")
	}
	if len(fb) > 0 {
		return fb, nil
	}

	errs, err = as.TryerEmail.Try(func() error {
		subject := "Confirm your StoryDevs handle"
		return as.sendConfirmationEmail("reserve", email, subject, code)
	})
	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsEmail, len(errs)).
			Data(sd.LK_ReserveId, id).
			Data(sd.LK_ReserveHandle, handle).
			Data(sd.LK_ReserveEmail, email)
		return nil, errors.New("Unable to send mailing list confirmation email.")
	}

	log.Info(reqId, "Created reservation.").
		Data(sd.LK_ReserveId, id).
		Data(sd.LK_ReserveHandle, handle).
		Data(sd.LK_ReserveEmail, email).
		Data(sd.LK_ReserveCode, code)

	return fb, nil
}

func (as *Account) ConfirmSubscription(reqId, code string) (bool, error) {

	tmp := struct {
		Id    int64
		Email string
	}{}

	errs, err := as.TryerTx.Try(func() error {

		tx, err := as.Db.Begin()
		if err != nil {
			return err
		}

		err = tx.Get(&tmp, `
			SELECT
				id,
				email
			FROM
				mailing
			WHERE
				code = $1`,
			code)
		if err != nil {
			return tx.Rollback(err)
		}

		_, err = tx.Exec(`
			UPDATE
				mailing
			SET
				code = NULL
			WHERE
				code = $1`,
			code)
		if err != nil {
			return tx.Rollback(err)
		}

		return tx.Commit()
	})

	log := as.Logger

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_SubscriptionCode, code)
		if lastErrSqlNoRows(errs) {
			return false, nil
		}
		return false, errors.New("Unable to confirm subscription.")
	}

	log.Info(reqId, "Confirmed subscription.").
		Data(sd.LK_SubscriptionId, tmp.Id).
		Data(sd.LK_SubscriptionEmail, tmp.Email)

	return true, nil
}

func (as *Account) ConfirmReservation(reqId, code string) (bool, error) {

	tmp := struct {
		Id     string
		Handle string
		Email  string
	}{}

	errs, err := as.TryerTx.Try(func() error {

		tx, err := as.Db.Begin()
		if err != nil {
			return err
		}

		err = tx.Get(&tmp, `
			SELECT
				id,
				email,
				handle
			FROM
				reserved
			WHERE
				code = $1`,
			code)
		if err != nil {
			return tx.Rollback(err)
		}

		_, err = tx.Exec(`
			UPDATE
				reserved
			SET
				code = NULL
			WHERE
				code = $1`,
			code)
		if err != nil {
			return tx.Rollback(err)
		}

		return tx.Commit()
	})

	log := as.Logger

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_ReserveCode, code)
		if lastErrSqlNoRows(errs) {
			return false, nil
		}
		return false, errors.New("Unable to confirm account.")
	}

	log.Info(reqId, "Confirmed reservation.").
		Data(sd.LK_ReserveId, tmp.Id).
		Data(sd.LK_ReserveEmail, tmp.Email).
		Data(sd.LK_ReserveHandle, tmp.Handle)

	return true, nil
}

func lastErrSqlNoRows(errs []error) bool {
	if len(errs) == 0 {
		return false
	}
	return errors.Is(errs[len(errs)-1], sql.ErrNoRows)
}

func (as *Account) ConfirmAccount(reqId, code string) (bool, error) {

	tmp := struct {
		AccId  string
		PId    string
		Handle string
		Email  string
	}{}

	errs, err := as.TryerTx.Try(func() error {

		tx, err := as.Db.Begin()
		if err != nil {
			return err
		}

		err = tx.Get(&tmp, `
			SELECT
				accounts.id AS accid,
				personas.id AS pid,
				accounts.email,
				personas.handle
			FROM
				accounts,
				personas
			WHERE
				accounts.code = $1 AND
				accounts.id = personas.acc_id`,
			code)
		if err != nil {
			return tx.Rollback(err)
		}

		_, err = tx.Exec(`
			UPDATE
				reserved
			SET
				redeemed = true
			WHERE
				handle ILIKE $1 AND
				email ILIKE $2
			`,
			tmp.Handle, tmp.Email)
		if err != nil {
			return tx.Rollback(err)
		}

		_, err = tx.Exec(`
			UPDATE
				accounts
			SET
				code = NULL
			WHERE
				code = $1`,
			code)
		if err != nil {
			return tx.Rollback(err)
		}

		return tx.Commit()
	})

	log := as.Logger

	if err != nil {
		log.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_RetryAttemptsTx, len(errs)).
			Data(sd.LK_AccCode, code)
		if lastErrSqlNoRows(errs) {
			return false, nil
		}
		return false, errors.New("Unable to confirm account.")
	}

	log.Info(reqId, "Confirmed account.").
		Data(sd.LK_AccId, tmp.AccId).
		Data(sd.LK_AccEmail, tmp.Email).
		Data(sd.LK_PersId, tmp.PId).
		Data(sd.LK_PersHandle, tmp.Handle)

	return true, nil
}

func (as *Account) Delete(reqId string, id int64) error {

	var handles []string
	var email string

	errs, err := as.TryerTx.Try(func() error {

		tx, err := as.Db.Begin()
		if err != nil {
			return err
		}

		err = tx.Select(&handles, `
			SELECT
				handle
			FROM
				personas
			WHERE
				acc_id = $1`,
			id)
		if err != nil {
			return tx.Rollback(err)
		}

		err = tx.Get(&email, `
			UPDATE
				accounts
			SET
				deleted = true
			WHERE
				id = $1
			RETURNING
				email`,
			id)
		if err != nil {
			return tx.Rollback(err)
		}

		return tx.Commit()
	})

	if err != nil {
		as.Logger.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs).
			Data(sd.LK_AccId, id)
		return err
	}

	e := as.Logger.Info(reqId, "Flagged account as deleted.")
	e.Data(sd.LK_AccId, id)
	e.Data(sd.LK_AccEmail, email)
	for _, h := range handles {
		e.Data(sd.LK_PersHandle, h)
	}

	return nil
}

func (as *Account) DeletePersona(reqId, token, slug string, accId int64) (newDefault string, err error) {

	errs, err := as.TryerTx.Try(func() error {

		tx, err := as.Db.Begin()
		if err != nil {
			return err
		}

		/*
			Check there's more than one persona
			that isn't flagged as deleted.
		*/
		var count int
		err = tx.Get(&count, `
			SELECT
				COUNT(*)
			FROM
				personas
			WHERE
				personas.acc_id = $1 AND
				personas.deleted IS NULL`,
			accId)
		if err != nil {
			return tx.Rollback(err)
		}
		if count < 2 {
			msg := "cannot delete an account's final persona"
			return tx.Rollback(errors.New(msg))
		}

		/*
			If the persona being deleted is the default one
			make the oldest remaining persona the new default.
		*/
		_, err = tx.Exec(`
			UPDATE
				personas
			SET
				default_p = true
			WHERE
			
				-- If we're deleting the default persona...
				(
					SELECT
						default_p
					FROM
						personas
					WHERE
						slug = $1
				)
				
				AND
				
				-- ...get the oldest non-default persona.
				created = (
					SELECT
						MIN(created)
					FROM
						personas
					WHERE
						slug      != $1   AND
						acc_id     = $2   AND
						deleted   != true AND
						default_p != true
				)`,
			slug, accId)
		if err != nil {
			return tx.Rollback(err)
		}

		/*
			Any logins set to the persona being deleted
			are updated to a new default persona. Because
			of the above query the default persona is now
			guaranteed not to be the one we're deleting.
		*/
		_, err = tx.Exec(`
			UPDATE
				logins
			SET
				p_id = (
					SELECT
						id
					FROM
						personas
					WHERE
						acc_id    = $3 AND
						default_p = true
				)
			FROM
				personas
			WHERE
				logins.token  = $1 AND
				logins.p_id   = personas.id AND
				personas.slug = $2`,
			token, slug, accId)
		if err != nil {
			return tx.Rollback(err)
		}

		_, err = tx.Exec(`
			UPDATE
				personas
			SET
				default_p = false,
				deleted   = true
			WHERE
				slug = $1`,
			slug)
		if err != nil {
			return tx.Rollback(err)
		}

		err = tx.Get(&newDefault, `
			SELECT
				personas.slug
			FROM
				personas,
				logins
			WHERE
				logins.token = $1 AND
				logins.p_id = personas.id`,
			token)
		if err != nil {
			return tx.Rollback(err)
		}

		return tx.Commit()
	})

	if err != nil {
		as.Logger.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs)
		return "", err
	}

	as.Logger.Info(reqId, "Flagged persona as deleted.").
		Data(sd.LK_AccId, accId).
		Data(sd.LK_PersSlug, slug)

	return newDefault, nil
}

func (as *Account) Switch(reqId, token, slug string) error {

	var pId int64

	errs, err := as.TryerTx.Try(func() error {

		tx, err := as.Db.Begin()
		if err != nil {
			return err
		}

		err = tx.Get(&pId, `
			SELECT
				id
			FROM
				personas
			WHERE
				slug = $1`,
			slug)
		if err != nil {
			return tx.Rollback(err)
		}

		_, err = tx.Exec(`
			UPDATE
				logins
			SET
				p_id = $1
			WHERE
				token = $2`,
			pId, token)
		if err != nil {
			return tx.Rollback(err)
		}

		return tx.Commit()
	})

	if err != nil {
		as.Logger.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs)
		return err
	}

	as.Logger.Info(reqId, "Switched persona").
		Data(sd.LK_PersId, pId)

	return nil
}

func (as *Account) Login(reqId, identity, pass string) (*sd.AuthedUser, error) {

	var account *sd.Account
	var err error
	var isEmail = strings.Contains(identity, "@")

	o := sd.AccOptRetrieve{
		Confirmed: true,
		Password:  true,
	}
	if isEmail {
		account, err = as.RetrieveByEmail(reqId, identity, o)
	} else {
		account, err = as.RetrieveByHandle(reqId, identity, o)
	}
	if err == sql.ErrNoRows {
		return nil, sd.ErrInvalidLoginAttempt
	}
	if err != nil {
		return nil, err
	}

	ok, err := as.Password.Compare(pass, account.Pass)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, sd.ErrInvalidLoginAttempt
	}

	activePersona := account.ActivePersona()

	token, err := newToken()
	if err != nil {
		return nil, err
	}

	errs, err := as.TryerTx.Try(func() error {

		tx, err := as.Db.Begin()
		if err != nil {
			return err
		}

		result, err := tx.Exec(`
			INSERT INTO logins (
				acc_id,
				p_id,
				token,
				since
			)
			VALUES
				($1, $2, $3, $4)`,
			account.Id, activePersona.Id, token, time.Now().Unix())
		if err != nil {
			return tx.Rollback(err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return tx.Rollback(err)
		}
		if affected != 1 {
			return tx.Rollback(errors.New("unable to insert into logins"))
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		as.Logger.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs)
		return nil, err
	}

	as.Logger.Info(reqId, "Logged in user.").
		Data(sd.LK_AccId, account.Id).
		Data(sd.LK_AccEmail, account.Email).
		Data(sd.LK_PersId, activePersona.Id).
		Data(sd.LK_PersHandle, activePersona.Handle)

	return &sd.AuthedUser{
		Token:   token,
		Account: account,
	}, nil
}

func (as *Account) Logout(reqId, token string) (ok bool, err error) {

	log := as.Logger
	db := as.Db

	user := struct {
		AccId  int64
		PId    int64
		Handle string
	}{}

	err = db.Get(&user, `
		SELECT
			accounts.id AS accid,
			personas.id AS pid,
			personas.handle
		FROM
			accounts,
			personas,
			logins
		WHERE
			token=$1 AND
			logins.acc_id=accounts.id AND
			logins.p_id=personas.id`,
		token)
	if err != nil {
		return false, err
	}

	_, err = db.Exec(`DELETE FROM logins WHERE token=$1`, token)
	if err != nil {
		return false, err
	}

	log.Info(reqId, "Logged out user.").
		Data(sd.LK_AccId, user.AccId).
		Data(sd.LK_PersId, user.PId).
		Data(sd.LK_PersHandle, user.Handle)

	return true, nil
}

func (as *Account) RetrieveByToken(reqId, token string, o sd.AccOptRetrieve) (*sd.Account, error) {
	return as.retrieve(reqId, "token", token, o)
}

func (as *Account) RetrieveById(reqId string, id int64, o sd.AccOptRetrieve) (*sd.Account, error) {
	return as.retrieve(reqId, "id", id, o)
}

func (as *Account) RetrieveByEmail(reqId, email string, o sd.AccOptRetrieve) (*sd.Account, error) {
	return as.retrieve(reqId, "email", email, o)
}

func (as *Account) RetrieveByHandle(reqId, handle string, o sd.AccOptRetrieve) (*sd.Account, error) {
	return as.retrieve(reqId, "handle", handle, o)
}

func (as *Account) retrieve(reqId, kind string, arg interface{}, o sd.AccOptRetrieve) (*sd.Account, error) {

	deleted := ""
	if !o.DeletedAccount {
		deleted += ` AND accounts.deleted IS NULL`
	}

	fromWhere := ""
	switch kind {
	case "token":
		fromWhere = fmt.Sprintf(`,
				logins.p_id
			FROM
				logins,
				accounts
			WHERE
				logins.token = $1 AND
				logins.acc_id = accounts.id
				%s`,
			deleted)
	case "id":
		fromWhere = fmt.Sprintf(`
			FROM
				accounts
			WHERE
				id=$1
				%s`,
			deleted)
	case "email":
		fromWhere = fmt.Sprintf(`
			FROM
				accounts
			WHERE
				email=$1
				%s`,
			deleted)
	case "handle":
		fromWhere = fmt.Sprintf(`
			FROM
				accounts,
				personas
			WHERE
				personas.handle=$1 AND
				accounts.id=personas.acc_id
				%s`,
			deleted)
	}

	if o.Confirmed {
		fromWhere += ` AND accounts.code IS NULL`
	}

	selectPass := ""
	if o.Password {
		selectPass = "accounts.pass,"
	}

	tmp := struct {
		P_id int64
		sd.Account
	}{}
	var personas []sd.Persona
	var notFound bool

	errs, err := as.TryerTx.Try(func() error {

		tx, err := as.Db.BeginRead()
		if err != nil {
			return err
		}

		err = tx.Get(&tmp, fmt.Sprintf(`
			SELECT
				
				%s
				
				accounts.id,
				accounts.email,
				accounts.created
				
				%s
			`, selectPass, fromWhere), arg)
		if err != nil {
			if strings.HasPrefix(err.Error(), sql.ErrNoRows.Error()) {
				notFound = true
			}
			return tx.Rollback(err)
		}

		deleted := ""
		if !o.DeletedPersona {
			deleted += ` AND deleted IS NULL`
		}

		err = tx.Select(&personas, fmt.Sprintf(`
			SELECT
				id,
				handle,
				name,
				created,
				default_p,
				slug,
				avatar,
				deleted,
				visibility,
				admin
			FROM
				personas
			WHERE
				acc_id=$1
				%s
			ORDER BY
				created ASC`, deleted),
			tmp.Account.Id)
		if err != nil {
			return tx.Rollback(err)
		}

		for i := range personas {
			err = tx.Select(&personas[i].Pronouns, `
				SELECT
					pronoun
				FROM
					pronouns
				WHERE
					ref_id=$1;`,
				personas[i].Id)
			if err != nil {
				return tx.Rollback(err)
			}
		}

		if err := tx.Commit(); err != nil {
			return err
		}

		return nil
	})

	if notFound {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		as.Logger.ErrorMulti(reqId, err.Error(), sd.LK_Err, errs)
		return nil, err
	}

	/*
		If we're retrieving by token or handle the active persona
		will be the one possessing that token or handle. Otherwise
		the default persona will be considered active.
	*/
	switch kind {
	case "token":
		for i := range personas {
			if personas[i].Id == tmp.P_id {
				personas[i].Active = true
				break
			}
		}
	case "handle":
		for i := range personas {
			if personas[i].Handle == arg.(string) {
				personas[i].Active = true
				break
			}
		}
	}

	tmp.Account.Personas = personas

	return &tmp.Account, nil
}

func borderedBySpace(s string) bool {
	if len(s) != len(strings.TrimSpace(s)) {
		return true
	}
	return false
}

func (as *Account) sendConfirmationEmail(kind, email, subject, code string) error {

	var url string
	if as.Config.Dev {
		url = "http://localhost:" + as.Config.Port
	} else {
		url = "https://storydevs.com"
	}

	msg := fmt.Sprintf(`This is your confirmation code:

%s

Alternatively, follow this link:

%s/%s/%s`,
		code, url, kind, code)

	return as.Emailer.Send(email, subject, msg)
}
