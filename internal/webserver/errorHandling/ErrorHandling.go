package errorHandling

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/forceu/gokapi/internal/helper"
	"github.com/forceu/gokapi/internal/languages"
)

var tokens = make(map[string]DisplayedError)
var mutex sync.RWMutex
var cleanupOnce sync.Once

const ttl = 5 * time.Minute

const WidthDefault = "20rem"
const WidthWide = "30rem"
const WidthVeryWide = "65%"

const (
	TypeFileNotFound = iota
	TypeInvalidFileRequest
	TypeE2ECipher
	TypeOAuthNotAuthorised
	TypeOAuthNonGeneric
)

type DisplayedError struct {
	// Title is a title that is displayed as-is and is not translated. It is only
	// used for values that are provided by an external system, e.g. an OIDC
	// error code. Otherwise, TitleKey should be used.
	Title string
	// TitleKey is the translation key of the title. If it is set, it has
	// priority over Title.
	TitleKey string
	// Message is a message that is displayed as-is and is not translated
	Message string
	// MessageKey is the translation key of the message. If it is set, it has
	// priority over Message.
	MessageKey string
	// MessageArgs are passed to the translated message and replace the
	// placeholders {0}, {1}, ...
	MessageArgs []string
	// MessageDetail is a technical detail, e.g. the text of an error, that is
	// appended to the translated message and is never translated itself
	MessageDetail string
	// OAuthProviderMessage is the error description that the OIDC provider sent
	OAuthProviderMessage string
	CardWidth            string
	ErrorId              int
	IsGeneric            bool
	expiry               int64
}

func (d DisplayedError) IsExpired() bool {
	return d.expiry < time.Now().Unix()
}

// GetTitle returns the title of the error in the language of the passed translator
func (d DisplayedError) GetTitle(translator languages.Translator) string {
	if d.TitleKey == "" {
		return d.Title
	}
	return translator.T(d.TitleKey)
}

// GetMessage returns the message of the error in the language of the passed translator
func (d DisplayedError) GetMessage(translator languages.Translator) string {
	message := d.Message
	if d.MessageKey != "" {
		args := make([]any, len(d.MessageArgs))
		for i, arg := range d.MessageArgs {
			args[i] = arg
		}
		message = translator.Tf(d.MessageKey, args...)
	}
	if d.MessageDetail != "" {
		if message == "" {
			return d.MessageDetail
		}
		return message + " " + d.MessageDetail
	}
	return message
}

// RedirectToErrorPage redirects to an error page that displays a translated title
// and message. The optional detail is appended to the message and is not translated.
func RedirectToErrorPage(w http.ResponseWriter, r *http.Request, titleKey, messageKey, messageDetail, cardWidth string) {
	result := DisplayedError{
		TitleKey:      titleKey,
		MessageKey:    messageKey,
		MessageDetail: messageDetail,
		expiry:        time.Now().Add(ttl).Unix(),
		CardWidth:     cardWidth,
	}
	redirectToError(w, r, result)
}

func RedirectGenericErrorPage(w http.ResponseWriter, r *http.Request, genericType int) {
	var cardWidth string
	switch genericType {
	case TypeFileNotFound:
		cardWidth = WidthDefault
	case TypeInvalidFileRequest:
		cardWidth = WidthWide
	case TypeE2ECipher:
		cardWidth = WidthVeryWide
	case TypeOAuthNotAuthorised:
		cardWidth = WidthWide
	default:
		redirectToError(w, r, DisplayedError{
			TitleKey:    "errorpage_unknown_title",
			MessageKey:  "errorpage_unknown_text",
			MessageArgs: []string{strconv.Itoa(genericType)},
			CardWidth:   WidthWide,
			expiry:      time.Now().Add(ttl).Unix(),
		})
		return
	}

	result := DisplayedError{
		expiry:    time.Now().Add(ttl).Unix(),
		ErrorId:   genericType,
		IsGeneric: true,
		CardWidth: cardWidth,
	}
	redirectToError(w, r, result)
}

// RedirectToOAuthErrorPage redirects to an error page for an OIDC error.
// messageKey is the translation key of the message, the text of the optional
// error is appended to it and is not translated.
func RedirectToOAuthErrorPage(w http.ResponseWriter, r *http.Request, messageKey string, err error) {
	if r.URL.Query().Get("error") == "access_denied" {
		result := DisplayedError{
			TitleKey:   "errorpage_access_denied_title",
			MessageKey: "errorpage_access_denied_text",
			expiry:     time.Now().Add(ttl).Unix(),
			ErrorId:    TypeOAuthNonGeneric,
			IsGeneric:  false,
		}
		redirectToError(w, r, result)
		return
	}
	var detail string
	if err != nil {
		detail = err.Error()
	}
	result := DisplayedError{
		Title:                r.URL.Query().Get("error"),
		MessageKey:           messageKey,
		MessageDetail:        detail,
		OAuthProviderMessage: r.URL.Query().Get("error_description"),
		expiry:               time.Now().Add(ttl).Unix(),
		ErrorId:              TypeOAuthNonGeneric,
		IsGeneric:            false,
	}
	redirectToError(w, r, result)
}

func redirectToError(w http.ResponseWriter, r *http.Request, displayedError DisplayedError) {
	token := helper.GenerateRandomString(30)
	mutex.Lock()
	tokens[token] = displayedError
	mutex.Unlock()

	cleanupOnce.Do(func() {
		go cleanup(true)
	})
	http.Redirect(w, r, "./error?e="+token, http.StatusTemporaryRedirect)
}

func Get(r *http.Request) DisplayedError {
	mutex.RLock()
	defer mutex.RUnlock()
	if !r.URL.Query().Has("e") {
		return DisplayedError{
			IsGeneric: true,
			ErrorId:   TypeFileNotFound,
			CardWidth: WidthDefault,
		}
	}
	displayedError, ok := tokens[r.URL.Query().Get("e")]
	if !ok {
		return DisplayedError{
			TitleKey:   "errorpage_unknown_id_title",
			MessageKey: "errorpage_unknown_id_text",
			CardWidth:  WidthDefault,
		}
	}
	return displayedError
}

func cleanup(periodic bool) {
	mutex.Lock()
	for id, token := range tokens {
		if token.IsExpired() {
			delete(tokens, id)
		}
	}
	mutex.Unlock()
	if periodic {
		time.Sleep(time.Hour)
		go cleanup(true)
	}

}
