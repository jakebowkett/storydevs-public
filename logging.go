package storydevs

import (
	"github.com/jakebowkett/go-logger/logger"
)

type Logger interface {
	Redirect(reqId string, code int)

	// These methods both log and write as responses their respective HTTP status.
	BadRequest(reqId string, w logger.HeaderWriter, msg string) *logger.Entry
	Unauthorised(reqId string, w logger.HeaderWriter)
	NotFound(reqId string, w logger.HeaderWriter)
	HttpStatus(reqId string, w logger.HeaderWriter, code int)

	ErrorMulti(reqId, msg, key string, errs []error) *logger.Entry

	Fatal(err error)

	Once(msg string)
	OnceF(format string, a ...interface{})

	Info(reqId, msg string) *logger.Entry
	Error(reqId, msg string) *logger.Entry
	Debug(reqId, msg string) *logger.Entry

	InfoF(reqId, format string, a ...interface{}) *logger.Entry
	ErrorF(reqId, format string, a ...interface{}) *logger.Entry
	DebugF(reqId, format string, a ...interface{}) *logger.Entry

	End(reqId, ip, method, route string, duration int64)

	Sess(name string) *logger.Session

	SetDebug(enabled bool)
	SetRuntime(enabled bool)
}

const (
	// Meta.
	LK_Version = "Version"

	// General context.
	LK_Msg = "Message"
	LK_Err = "Error"

	// Actions.
	LK_EmailAttempt = "Attempt To Send Email"
	LK_EmailSent    = "Email Sent"

	// Data.
	LK_RetryAttemptsDisk  = "Disk Attempts"
	LK_RetryAttemptsTx    = "Transaction Attempts"
	LK_RetryAttemptsEmail = "Email Attempts"

	LK_FileName = "File Name"

	LK_UserToken    = "User Token"
	LK_UserIdentity = "User Identity"

	LK_SubscriptionCode  = "Subscription Code"
	LK_SubscriptionId    = "Subscription Id"
	LK_SubscriptionEmail = "Subscription Email"

	LK_ReserveCode   = "Reserve Code"
	LK_ReserveId     = "Reserve Id"
	LK_ReserveEmail  = "Reserve Email"
	LK_ReserveHandle = "Reserve Handle"

	LK_InviteCode = "Invite Code"
	LK_InviteId   = "Invite Id"
	LK_InviteSlug = "Invite Slug"

	LK_AccCode       = "Account Code"
	LK_AccId         = "Account Id"
	LK_AccEmail      = "Account Email"
	LK_AccEmailOld   = "Account Old Email"
	LK_AccEmailNew   = "Account New Email"
	LK_AccEmailCode  = "Account Email Code"
	LK_AccPassCode   = "Account Password Change Code"
	LK_AccForgotCode = "Account Password Forgot Code"

	LK_PersSlug   = "Persona Slug"
	LK_PersHandle = "Persona Handle"
	LK_PersId     = "Persona Id"

	LK_Mode         = "Mode"
	LK_ResourceId   = "Resource Id"
	LK_ResourceSlug = "Resource Slug"

	LK_LibraryId    = "Library Id"
	LK_LibraryTitle = "Library Title"
	LK_LibrarySlug  = "Library Slug"
	LK_LibraryWords = "Library Word Count"

	LK_PostId      = "Post Id"
	LK_PostSlug    = "Post Id"
	LK_PostWords   = "Post Word Count"
	LK_ThreadId    = "Thread Id"
	LK_ThreadSlug  = "Thread Slug"
	LK_ThreadTitle = "Thread Title"
	LK_ThreadHead  = "Thread Head"

	LK_EventId   = "Event Id"
	LK_EventName = "Event Name"
	LK_EventSlug = "Event Slug"

	LK_ProfileId   = "Profile Id"
	LK_ProfileName = "Profile Name"
	LK_ProfileSlug = "Profile Slug"

	LK_Emailer      = "Emailer State"
	LK_EmailRcpt    = "Email Recipient"
	LK_EmailSubject = "Email Subject"
)
