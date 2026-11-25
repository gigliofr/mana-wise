package notifications

import (
	"strings"

	"github.com/gigliofr/gokit/mailx"
	"github.com/gigliofr/mana-wise/domain"
)

// mailOptions sono le opzioni comuni di ManaWise per gokit/mailx.
var mailOptions = mailx.Options{DefaultSubject: "ManaWise notification"}

// EmailSenderConfigStatus indica se l'invio email è configurato e perché.
// Resend (RESEND_API_KEY + RESEND_FROM_EMAIL) ha la precedenza su SMTP
// (SMTP_HOST, SMTP_PORT, SMTP_USER, SMTP_KEY, MAIL_FROM).
func EmailSenderConfigStatus() (bool, string) {
	s, why := mailx.FromEnv(mailOptions)
	_, noop := s.(mailx.Noop)
	return !noop, why
}

// NewEmailSenderFromEnv costruisce il mittente email da gokit/mailx;
// senza configurazione restituisce un sender che non invia nulla.
func NewEmailSenderFromEnv() domain.EmailSender {
	s, _ := mailx.FromEnv(mailOptions)
	return s
}

// stripHTMLBasic resta per i template esistenti.
func stripHTMLBasic(v string) string { return strings.TrimSpace(mailx.HTMLToText(v)) }
