package notifications

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/gigliofr/mana-wise/domain"
	"github.com/resend/resend-go/v2"
)

type resendSender struct {
	client *resend.Client
	from   string
}

func resendConfigFromEnv() (string, string, bool) {
	apiKey := strings.TrimSpace(os.Getenv("RESEND_API_KEY"))
	from := strings.TrimSpace(os.Getenv("RESEND_FROM_EMAIL"))
	return apiKey, from, apiKey != "" && from != ""
}

// ResendEmailSenderConfigStatus returns whether Resend is configured and a diagnostic reason.
func ResendEmailSenderConfigStatus() (bool, string) {
	apiKey, from, ok := resendConfigFromEnv()
	if !ok {
		missing := []string{}
		if apiKey == "" {
			missing = append(missing, "RESEND_API_KEY")
		}
		if from == "" {
			missing = append(missing, "RESEND_FROM_EMAIL")
		}
		return false, "missing env vars: " + strings.Join(missing, ",")
	}
	return true, "resend configured"
}

// NewResendEmailSender creates a Resend email sender.
// Returns NoopEmailSender if Resend is not properly configured.
func NewResendEmailSender() domain.EmailSender {
	apiKey, from, ok := resendConfigFromEnv()
	if !ok {
		return domain.NoopEmailSender{}
	}

	return &resendSender{
		client: resend.NewClient(apiKey),
		from:   from,
	}
}

func (s *resendSender) Send(to, subject, textBody, htmlBody string) error {
	to = strings.TrimSpace(to)
	if to == "" {
		return errors.New("recipient email required")
	}

	if strings.TrimSpace(subject) == "" {
		subject = "ManaWise notification"
	}

	if strings.TrimSpace(textBody) == "" && strings.TrimSpace(htmlBody) == "" {
		return errors.New("email body is empty")
	}

	// If only text body provided, generate minimal HTML
	if strings.TrimSpace(htmlBody) == "" {
		htmlBody = "<p>" + strings.ReplaceAll(textBody, "\n", "<br>") + "</p>"
	}

	// If only HTML body provided, strip tags for text version
	if strings.TrimSpace(textBody) == "" {
		textBody = stripHTMLBasic(htmlBody)
	}

	// Prepare Resend email request
	req := &resend.SendEmailRequest{
		From:    s.from,
		To:      []string{to},
		Subject: subject,
		Text:    textBody,
		Html:    htmlBody,
	}

	// Send email via Resend
	sent, err := s.client.Emails.Send(req)
	if err != nil {
		return fmt.Errorf("resend send failed: %w", err)
	}

	// Check if email was successfully sent
	if sent == nil || sent.Id == "" {
		return errors.New("resend send failed: no email ID returned")
	}

	return nil
}
