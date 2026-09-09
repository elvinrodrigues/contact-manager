package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const mailerTimeout = 10 * time.Second

// IsDebugEmail returns true when DEBUG_EMAIL=true, which skips Resend and logs
// that an email would have been sent. It never logs the body: verification and
// reset links contain the raw token, and those are stored hashed precisely so
// that reading them back out of a log is not possible.
func IsDebugEmail() bool {
	return strings.EqualFold(os.Getenv("DEBUG_EMAIL"), "true")
}

// SendEmail dispatches an HTML email via the Resend API. It never returns an
// error — failures are logged so they cannot break signup or reset flows. The
// context lets a shutdown cancel an in-flight send.
func SendEmail(ctx context.Context, to, subject, html string) {
	if IsDebugEmail() {
		log.Printf("[MAILER DEBUG] would send %q to %s (body withheld: contains a secret token)",
			subject, maskEmail(to))
		return
	}

	// Optional dev override: redirect all mail to one inbox. The target must be
	// supplied explicitly; there is no default recipient.
	if strings.EqualFold(os.Getenv("FORCE_EMAIL"), "true") {
		override := strings.TrimSpace(os.Getenv("FORCE_EMAIL_TO"))
		if override == "" {
			log.Printf("[MAILER ERROR] FORCE_EMAIL=true but FORCE_EMAIL_TO is unset — refusing to send")
			return
		}
		log.Printf("[MAILER] Redirecting %s -> %s (FORCE_EMAIL)", maskEmail(to), maskEmail(override))
		to = override
	}

	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		log.Printf("[MAILER ERROR] RESEND_API_KEY not set — cannot send email to %s", maskEmail(to))
		return
	}

	from := os.Getenv("MAIL_FROM")
	if from == "" {
		from = "ContactHub <onboarding@resend.dev>"
	}

	payload := map[string]interface{}{
		"from":    from,
		"to":      []string{to},
		"subject": subject,
		"html":    html,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[MAILER ERROR] Failed to marshal payload: %v", err)
		return
	}

	sendCtx, cancel := context.WithTimeout(ctx, mailerTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(sendCtx, http.MethodPost,
		"https://api.resend.com/emails", bytes.NewReader(jsonData))
	if err != nil {
		log.Printf("[MAILER ERROR] Failed to create request: %v", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	log.Printf("[MAILER] Sending %q to %s", subject, maskEmail(to))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("[MAILER ERROR] HTTP request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		// Cap the echoed body: it is a third party's response, not ours.
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		log.Printf("[MAILER ERROR] Resend returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
		return
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	log.Printf("[MAILER OK] Resend accepted message for %s (status %d)", maskEmail(to), resp.StatusCode)
}

func maskEmail(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || len(parts[0]) == 0 {
		return "***@***"
	}
	if len(parts[0]) <= 2 {
		return "***@" + parts[1]
	}
	return parts[0][:2] + "***@" + parts[1]
}

// GetBaseURL reads the frontend base URL from the environment.
func GetBaseURL() string {
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	return strings.TrimSuffix(baseURL, "/")
}
