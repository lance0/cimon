// Package notify provides cross-platform desktop notifications and hook execution for cimon.
package notify

import (
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
)

// NotificationData contains information for desktop notifications
type NotificationData struct {
	WorkflowName string
	RunNumber    int
	Conclusion   string // success, failure, cancelled, etc.
	Repo         string
	Branch       string
	HTMLURL      string
}

// NotifyResult contains the result of a notification attempt
type NotifyResult struct {
	Sent  bool
	Error error
}

// SendDesktopNotification sends an OS-native desktop notification.
// The notification is sent asynchronously (fire and forget).
func SendDesktopNotification(data NotificationData) NotifyResult {
	title := formatTitle(data)
	body := formatBody(data)
	urgency := getUrgency(data.Conclusion)

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = buildLinuxNotification(title, body, urgency)
	case "darwin":
		cmd = buildMacOSNotification(title, body)
	case "windows":
		cmd = buildWindowsNotification(title, body)
	default:
		return NotifyResult{Sent: false, Error: fmt.Errorf("unsupported platform: %s", runtime.GOOS)}
	}

	if cmd == nil {
		return NotifyResult{Sent: false, Error: fmt.Errorf("failed to build notification command")}
	}

	// Start the command asynchronously (non-blocking)
	if err := cmd.Start(); err != nil {
		return NotifyResult{Sent: false, Error: err}
	}

	// Wait for completion in background goroutine
	go func() {
		_ = cmd.Wait()
	}()

	return NotifyResult{Sent: true, Error: nil}
}

// formatTitle creates the notification title
func formatTitle(data NotificationData) string {
	icon := getStatusIcon(data.Conclusion)
	return fmt.Sprintf("%s %s #%d", icon, data.WorkflowName, data.RunNumber)
}

// formatBody creates the notification body
func formatBody(data NotificationData) string {
	conclusion := data.Conclusion
	if conclusion == "" {
		conclusion = "completed"
	}
	return fmt.Sprintf("%s on %s - %s", data.Repo, data.Branch, conclusion)
}

// getStatusIcon returns an emoji for the conclusion status
func getStatusIcon(conclusion string) string {
	switch conclusion {
	case "success":
		return "✓"
	case "failure":
		return "✗"
	case "cancelled":
		return "⊘"
	case "timed_out":
		return "⏱"
	default:
		return "●"
	}
}

// getUrgency returns the notification urgency level based on conclusion
func getUrgency(conclusion string) string {
	switch conclusion {
	case "failure", "timed_out":
		return "critical"
	case "cancelled":
		return "normal"
	default:
		return "low"
	}
}

// buildLinuxNotification builds a notify-send command for Linux
func buildLinuxNotification(title, body, urgency string) *exec.Cmd {
	// Check if notify-send is available
	if _, err := exec.LookPath("notify-send"); err != nil {
		return nil
	}
	return exec.Command("notify-send",
		"-u", urgency,
		"-a", "cimon",
		"-i", "dialog-information",
		title,
		body,
	)
}

// buildMacOSNotification builds an osascript command for macOS
func buildMacOSNotification(title, body string) *exec.Cmd {
	script := fmt.Sprintf(`display notification "%s" with title "%s" sound name "default"`, body, title)
	return exec.Command("osascript", "-e", script)
}

// buildWindowsNotification builds a PowerShell command for Windows toast notification
func buildWindowsNotification(title, body string) *exec.Cmd {
	// PowerShell script for Windows toast notification
	ps := fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null

$template = @"
<toast>
    <visual>
        <binding template="ToastText02">
            <text id="1">%s</text>
            <text id="2">%s</text>
        </binding>
    </visual>
</toast>
"@

$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml($template)
$toast = [Windows.UI.Notifications.ToastNotification]::new($xml)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("cimon").Show($toast)
`, title, body)

	return exec.Command("powershell", "-WindowStyle", "Hidden", "-Command", ps)
}

// IsNotificationAvailable checks if desktop notifications are supported on this platform
func IsNotificationAvailable() bool {
	switch runtime.GOOS {
	case "linux":
		_, err := exec.LookPath("notify-send")
		return err == nil
	case "darwin":
		// osascript is always available on macOS
		return true
	case "windows":
		// PowerShell is always available on modern Windows
		return true
	default:
		return false
	}
}

// HookData contains information passed to hook scripts via environment variables
type HookData struct {
	WorkflowName string
	RunNumber    int
	RunID        int64
	Status       string
	Conclusion   string
	Repo         string
	Branch       string
	Event        string
	Actor        string
	HTMLURL      string
	JobCount     int
	SuccessCount int
	FailureCount int
}

// ToEnvVars converts HookData to a slice of environment variable strings
func (h HookData) ToEnvVars() []string {
	return []string{
		"CIMON_WORKFLOW_NAME=" + h.WorkflowName,
		"CIMON_RUN_NUMBER=" + strconv.Itoa(h.RunNumber),
		"CIMON_RUN_ID=" + strconv.FormatInt(h.RunID, 10),
		"CIMON_STATUS=" + h.Status,
		"CIMON_CONCLUSION=" + h.Conclusion,
		"CIMON_REPO=" + h.Repo,
		"CIMON_BRANCH=" + h.Branch,
		"CIMON_EVENT=" + h.Event,
		"CIMON_ACTOR=" + h.Actor,
		"CIMON_HTML_URL=" + h.HTMLURL,
		"CIMON_JOB_COUNT=" + strconv.Itoa(h.JobCount),
		"CIMON_SUCCESS_COUNT=" + strconv.Itoa(h.SuccessCount),
		"CIMON_FAILURE_COUNT=" + strconv.Itoa(h.FailureCount),
	}
}
