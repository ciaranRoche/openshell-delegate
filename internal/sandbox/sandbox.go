package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// Create creates an OpenShell sandbox from the given image with the specified
// policy and launch command. It blocks until the creation command returns.
func Create(name, image, policy, provider, launchCmd string, uploads map[string]string) error {
	args := []string{
		"sandbox", "create",
		"--name", name,
		"--from", image,
		"--policy", policy,
	}

	if provider != "" {
		args = append(args, "--provider", provider)
	}

	for src, dst := range uploads {
		args = append(args, "--upload", fmt.Sprintf("%s:%s", src, dst))
	}

	args = append(args, "--", "sh", "-c", launchCmd)

	cmd := exec.Command("openshell", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Start() // Don't wait -- sandbox runs in background
}

// WaitReady polls until the sandbox reaches Ready state or the timeout expires.
func WaitReady(name string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if IsReady(name) {
			return nil
		}
		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("sandbox %q did not reach Ready state within %s", name, timeout)
}

// IsReady checks if a sandbox is in Ready state.
func IsReady(name string) bool {
	cmd := exec.Command("openshell", "sandbox", "list")
	out, err := cmd.Output()
	if err != nil {
		return false
	}

	// Strip ANSI codes and check for name + Ready
	cleaned := stripAnsi(string(out))
	for _, line := range strings.Split(cleaned, "\n") {
		if strings.Contains(line, name) && strings.Contains(line, "Ready") {
			return true
		}
	}
	return false
}

// Exec runs a command inside a running sandbox and returns the output.
func Exec(name string, command string) (string, error) {
	cmd := exec.Command("openshell", "sandbox", "exec", "-n", name, "--no-tty", "--", "bash", "-c", command)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("sandbox exec failed: %w\n%s", err, string(out))
	}
	return string(out), nil
}

// Upload transfers a local path to the sandbox.
func Upload(name, localPath, remotePath string) error {
	cmd := exec.Command("openshell", "sandbox", "upload", name, localPath, remotePath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}
	return nil
}

// Download transfers a remote path from the sandbox to the local filesystem.
func Download(name, remotePath, localPath string) error {
	cmd := exec.Command("openshell", "sandbox", "download", name, remotePath, localPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	return nil
}

// Delete removes a sandbox.
func Delete(name string) error {
	cmd := exec.Command("openshell", "sandbox", "delete", name)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// List returns the raw output of openshell sandbox list.
func List() (string, error) {
	cmd := exec.Command("openshell", "sandbox", "list")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// stripAnsi removes ANSI escape codes from a string.
func stripAnsi(s string) string {
	var result strings.Builder
	inEscape := false
	for _, r := range s {
		if r == '\033' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}
