package domain

// EmailSender sends transactional emails.
type EmailSender interface {
	Send(to, subject, textBody, htmlBody string) error
}

// NoopEmailSender is used when SMTP is not configured.
type NoopEmailSender struct{}

func (NoopEmailSender) Send(_, _, _, _ string) error { return nil }
