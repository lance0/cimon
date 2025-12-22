package notify

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// HookResult contains the result of a hook execution attempt
type HookResult struct {
	Executed bool
	Error    error
}

// ExecuteHook runs a user-specified script with workflow data as environment variables.
// The hook is executed asynchronously (fire and forget).
func ExecuteHook(hookPath string, data HookData) HookResult {
	if hookPath == "" {
		return HookResult{Executed: false, Error: fmt.Errorf("no hook path specified")}
	}

	// Resolve the hook path
	absPath, err := resolveHookPath(hookPath)
	if err != nil {
		return HookResult{Executed: false, Error: err}
	}

	// Check if the hook file exists and is executable
	if err := validateHookFile(absPath); err != nil {
		return HookResult{Executed: false, Error: err}
	}

	// Build the command
	cmd := buildHookCommand(absPath, data)
	if cmd == nil {
		return HookResult{Executed: false, Error: fmt.Errorf("failed to build hook command")}
	}

	// Start the command asynchronously (non-blocking)
	if err := cmd.Start(); err != nil {
		return HookResult{Executed: false, Error: fmt.Errorf("failed to start hook: %w", err)}
	}

	// Wait for completion in background goroutine
	go func() {
		_ = cmd.Wait()
	}()

	return HookResult{Executed: true, Error: nil}
}

// resolveHookPath resolves the hook path to an absolute path
func resolveHookPath(hookPath string) (string, error) {
	// If it's already absolute, use it directly
	if filepath.IsAbs(hookPath) {
		return hookPath, nil
	}

	// Try to resolve relative to current directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	absPath := filepath.Join(cwd, hookPath)
	return absPath, nil
}

// validateHookFile checks if the hook file exists and is executable
func validateHookFile(path string) error {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("hook file not found: %s", path)
	}
	if err != nil {
		return fmt.Errorf("failed to stat hook file: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("hook path is a directory, not a file: %s", path)
	}

	// On Unix-like systems, check if the file is executable
	if runtime.GOOS != "windows" {
		mode := info.Mode()
		if mode&0111 == 0 {
			return fmt.Errorf("hook file is not executable: %s (try: chmod +x %s)", path, path)
		}
	}

	return nil
}

// buildHookCommand creates the exec.Cmd for running the hook
func buildHookCommand(hookPath string, data HookData) *exec.Cmd {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		// On Windows, use cmd.exe to run the script
		ext := filepath.Ext(hookPath)
		switch ext {
		case ".ps1":
			cmd = exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", hookPath)
		case ".bat", ".cmd":
			cmd = exec.Command("cmd", "/C", hookPath)
		default:
			// Try to run directly (for .exe files or scripts with shebang)
			cmd = exec.Command(hookPath)
		}
	} else {
		// On Unix-like systems, run the script directly (relies on shebang)
		cmd = exec.Command(hookPath)
	}

	// Set environment variables with workflow data
	cmd.Env = append(os.Environ(), data.ToEnvVars()...)

	return cmd
}

// ValidateHookPath checks if a hook path is valid without executing it
func ValidateHookPath(hookPath string) error {
	if hookPath == "" {
		return nil // Empty path is valid (no hook configured)
	}

	absPath, err := resolveHookPath(hookPath)
	if err != nil {
		return err
	}

	return validateHookFile(absPath)
}
