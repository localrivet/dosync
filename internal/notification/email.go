package notification

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"os"
	"strings"
	"time"
)

// EmailNotifier implements the Notifier interface for email notifications
type EmailNotifier struct {
	BaseNotifier
	smtpClient *smtp.Client
	from       string
	auth       smtp.Auth
}

// NewEmailNotifier creates a new email notifier
func NewEmailNotifier(config NotificationConfig) *EmailNotifier {
	email := &EmailNotifier{}
	_ = email.Configure(config)
	return email
}

// Configure sets up the email notifier with the provided configuration
func (e *EmailNotifier) Configure(config NotificationConfig) error {
	err := e.BaseNotifier.Configure(config)
	if err != nil {
		return err
	}

	// Extract SMTP server and port
	serverParts := strings.Split(config.Endpoint, ":")
	if len(serverParts) != 2 {
		return fmt.Errorf("invalid SMTP server:port format: %s", config.Endpoint)
	}

	// Use token as sender address if not provided separately
	e.from = config.Token

	// Create SMTP client
	if e.smtpClient != nil {
		e.smtpClient.Quit()
		e.smtpClient = nil
	}

	// We don't actually connect here since we'll connect on-demand when sending
	// This is to avoid maintaining long-lived connections that might time out

	return nil
}

// connectSMTP establishes a connection to the SMTP server
func (e *EmailNotifier) connectSMTP() error {
	serverParts := strings.Split(e.Config.Endpoint, ":")
	server := serverParts[0]

	// Connect to the SMTP server
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // For testing; in production, use proper certificate validation
		ServerName:         server,
	}

	// Connect to the server
	conn, err := tls.Dial("tcp", e.Config.Endpoint, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %v", err)
	}

	// Create a new SMTP client
	client, err := smtp.NewClient(conn, server)
	if err != nil {
		return fmt.Errorf("failed to create SMTP client: %v", err)
	}

	// Authenticate if token (password) is provided
	if e.Config.Token != "" {
		// Hardcoded auth method; in production, this should be configurable
		auth := smtp.PlainAuth("", e.from, e.Config.Token, server)
		if err := client.Auth(auth); err != nil {
			client.Quit()
			return fmt.Errorf("SMTP authentication failed: %v", err)
		}
	}

	e.smtpClient = client
	return nil
}

// ShouldNotify checks if notifications should be sent based on the event type
func (e *EmailNotifier) ShouldNotify() bool {
	// This is a helper that checks all notification types
	// In a real implementation, we'd check based on the specific event
	return e.ShouldNotifyOnSuccess() || e.ShouldNotifyOnFailure() || e.ShouldNotifyOnRollback()
}

// sendEmail sends an email with the given subject and both HTML and text content
func (e *EmailNotifier) sendEmail(subject, htmlContent, textContent string) error {
	if len(e.Config.Recipients) == 0 {
		return fmt.Errorf("no recipients specified")
	}

	// Connect to SMTP server (on-demand)
	if e.smtpClient == nil {
		if err := e.connectSMTP(); err != nil {
			return err
		}
		defer e.smtpClient.Quit()
	}

	// Set the sender
	if err := e.smtpClient.Mail(e.from); err != nil {
		return fmt.Errorf("failed to set sender: %v", err)
	}

	// Set the recipients
	for _, recipient := range e.Config.Recipients {
		if err := e.smtpClient.Rcpt(recipient); err != nil {
			return fmt.Errorf("failed to add recipient %s: %v", recipient, err)
		}
	}

	// Start the data session
	w, err := e.smtpClient.Data()
	if err != nil {
		return fmt.Errorf("failed to start data session: %v", err)
	}

	// Construct the email headers
	boundary := "dosyncnotification"
	headers := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: multipart/alternative; boundary=%s\r\n\r\n",
		e.from, strings.Join(e.Config.Recipients, ", "), subject, boundary)

	// Construct the email body (multipart with text and HTML)
	body := headers +
		fmt.Sprintf("--%s\r\n", boundary) +
		"Content-Type: text/plain; charset=utf-8\r\n\r\n" +
		textContent + "\r\n\r\n" +
		fmt.Sprintf("--%s\r\n", boundary) +
		"Content-Type: text/html; charset=utf-8\r\n\r\n" +
		htmlContent + "\r\n\r\n" +
		fmt.Sprintf("--%s--", boundary)

	// Write the email to the data writer
	if _, err := w.Write([]byte(body)); err != nil {
		return fmt.Errorf("failed to write email body: %v", err)
	}

	// Close the data session
	if err := w.Close(); err != nil {
		return fmt.Errorf("failed to close data session: %v", err)
	}

	return nil
}

// getHostnameForEmail retrieves the hostname of the server
func getHostnameForEmail() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

// SendDeploymentStarted sends an email notification for a deployment start
func (e *EmailNotifier) SendDeploymentStarted(service, version string) error {
	if !e.ShouldNotifyOnSuccess() {
		return nil
	}

	hostname := getHostnameForEmail()
	subject := fmt.Sprintf("Deployment Started: %s to version %s on %s", service, version, hostname)

	textContent := fmt.Sprintf(
		"Deployment Started\n\n"+
			"Service: %s\n"+
			"Version: %s\n"+
			"Server: %s\n"+
			"Time: %s",
		service, version, hostname, time.Now().Format(time.RFC1123))

	htmlContent := fmt.Sprintf(
		"<html><body>"+
			"<h2>Deployment Started</h2>"+
			"<p><strong>Service:</strong> %s</p>"+
			"<p><strong>Version:</strong> %s</p>"+
			"<p><strong>Server:</strong> %s</p>"+
			"<p><strong>Time:</strong> %s</p>"+
			"</body></html>",
		service, version, hostname, time.Now().Format(time.RFC1123))

	return e.sendEmail(subject, htmlContent, textContent)
}

// SendDeploymentSuccess sends an email notification for a successful deployment
func (e *EmailNotifier) SendDeploymentSuccess(service, version string, duration time.Duration) error {
	if !e.ShouldNotifyOnSuccess() {
		return nil
	}

	hostname := getHostnameForEmail()
	subject := fmt.Sprintf("Deployment Successful: %s to version %s on %s", service, version, hostname)

	textContent := fmt.Sprintf(
		"Deployment Successful\n\n"+
			"Service: %s\n"+
			"Version: %s\n"+
			"Server: %s\n"+
			"Duration: %s\n"+
			"Time: %s",
		service, version, hostname, duration.String(), time.Now().Format(time.RFC1123))

	htmlContent := fmt.Sprintf(
		"<html><body>"+
			"<h2 style='color: green;'>Deployment Successful</h2>"+
			"<p><strong>Service:</strong> %s</p>"+
			"<p><strong>Version:</strong> %s</p>"+
			"<p><strong>Server:</strong> %s</p>"+
			"<p><strong>Duration:</strong> %s</p>"+
			"<p><strong>Time:</strong> %s</p>"+
			"</body></html>",
		service, version, hostname, duration.String(), time.Now().Format(time.RFC1123))

	return e.sendEmail(subject, htmlContent, textContent)
}

// SendDeploymentFailure sends an email notification for a failed deployment
func (e *EmailNotifier) SendDeploymentFailure(service, version, errorMsg string) error {
	if !e.ShouldNotifyOnFailure() {
		return nil
	}

	hostname := getHostnameForEmail()
	subject := fmt.Sprintf("Deployment Failed: %s to version %s on %s", service, version, hostname)

	textContent := fmt.Sprintf(
		"Deployment Failed\n\n"+
			"Service: %s\n"+
			"Version: %s\n"+
			"Server: %s\n"+
			"Error: %s\n"+
			"Time: %s",
		service, version, hostname, errorMsg, time.Now().Format(time.RFC1123))

	htmlContent := fmt.Sprintf(
		"<html><body>"+
			"<h2 style='color: red;'>Deployment Failed</h2>"+
			"<p><strong>Service:</strong> %s</p>"+
			"<p><strong>Version:</strong> %s</p>"+
			"<p><strong>Server:</strong> %s</p>"+
			"<p><strong>Error:</strong> %s</p>"+
			"<p><strong>Time:</strong> %s</p>"+
			"</body></html>",
		service, version, hostname, errorMsg, time.Now().Format(time.RFC1123))

	return e.sendEmail(subject, htmlContent, textContent)
}

// SendRollback sends an email notification for a rollback
func (e *EmailNotifier) SendRollback(service, fromVersion, toVersion string) error {
	if !e.ShouldNotifyOnRollback() {
		return nil
	}

	hostname := getHostnameForEmail()
	subject := fmt.Sprintf("Deployment Rolled Back: %s from %s to %s on %s", service, fromVersion, toVersion, hostname)

	textContent := fmt.Sprintf(
		"Deployment Rolled Back\n\n"+
			"Service: %s\n"+
			"Server: %s\n"+
			"From Version: %s\n"+
			"To Version: %s\n"+
			"Time: %s",
		service, hostname, fromVersion, toVersion, time.Now().Format(time.RFC1123))

	htmlContent := fmt.Sprintf(
		"<html><body>"+
			"<h2 style='color: orange;'>Deployment Rolled Back</h2>"+
			"<p><strong>Service:</strong> %s</p>"+
			"<p><strong>Server:</strong> %s</p>"+
			"<p><strong>From Version:</strong> %s</p>"+
			"<p><strong>To Version:</strong> %s</p>"+
			"<p><strong>Time:</strong> %s</p>"+
			"</body></html>",
		service, hostname, fromVersion, toVersion, time.Now().Format(time.RFC1123))

	return e.sendEmail(subject, htmlContent, textContent)
}
