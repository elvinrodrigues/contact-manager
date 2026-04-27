package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// IsDebugEmail returns true when DEBUG_EMAIL=true, which skips Resend and
// prints verification/reset links directly to the console for local testing.
func IsDebugEmail() bool {
	return strings.EqualFold(os.Getenv("DEBUG_EMAIL"), "true")
}

// SendEmail dispatches an HTML email via the Resend API.
// It never returns an error — failures are logged so they don't break signup/reset flows.
func SendEmail(to, subject, html string) {
	// ── Debug mode: print to console instead of calling Resend ───────────
	if IsDebugEmail() {
		log.Println("════════════════════════════════════════════════")
		log.Println("[MAILER DEBUG] Email would be sent:")
		log.Printf("  To:      %s", to)
		log.Printf("  Subject: %s", subject)
		log.Printf("  Body:    %s", html)
		log.Println("════════════════════════════════════════════════")
		return
	}
	// ── Force recipient override for dev/testing ────────────────────────
	if strings.EqualFold(os.Getenv("FORCE_EMAIL"), "true") {
		originalTo := to
		to = "elvinrodrigues3456@gmail.com"
		log.Printf("[MAILER] Original recipient: %s → Forced to: %s", maskEmail(originalTo), maskEmail(to))
	}

	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		log.Printf("[MAILER ERROR] RESEND_API_KEY not set — cannot send email to %s", maskEmail(to))
		return
	}

	payload := map[string]interface{}{
		"from":    "ContactHub <onboarding@resend.dev>",
		"to":      []string{to},
		"subject": subject,
		"html":    html,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[MAILER ERROR] Failed to marshal payload: %v", err)
		return
	}

	log.Printf("[MAILER] Sending email to %s | Subject: %s", maskEmail(to), subject)

	req, err := http.NewRequest("POST", "https://api.resend.com/emails", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("[MAILER ERROR] Failed to create request: %v", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("[MAILER ERROR] HTTP request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	bodyStr := string(bodyBytes)

	if resp.StatusCode >= 400 {
		log.Printf("[MAILER ERROR] Resend returned %d: %s", resp.StatusCode, bodyStr)
		return
	}

	log.Printf("[MAILER OK] Status %d | Response: %s", resp.StatusCode, bodyStr)
}

func maskEmail(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) != 2 || len(parts[0]) == 0 {
		return "***@***"
	}

	localPart := parts[0]
	if len(localPart) <= 2 {
		return "***@" + parts[1]
	}

	return localPart[:2] + "***@" + parts[1]
}

// GetBaseURL reads the frontend base URL from the environment, defaulting to localhost.
func GetBaseURL() string {
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:3000"
	}
	return strings.TrimSuffix(baseURL, "/")
}

