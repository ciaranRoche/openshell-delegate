package sandbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

// Upload transfers a local directory to the sandbox using a tarball to
// preserve dotfiles (e.g. .gitignore). The openshell directory upload
// strips dotfiles, so we tar locally, upload the archive as a single
// file, and extract it inside the sandbox.
func Upload(name, localPath, remotePath string) error {
	// Create a temp tarball of the directory contents.
	tmpFile, err := os.CreateTemp("", "od-upload-*.tar.gz")
	if err != nil {
		return fmt.Errorf("could not create temp file: %w", err)
	}
	tarPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tarPath)

	// Tar the directory contents (not the directory itself).
	// Use -C to change into the directory so paths are relative.
	tarCmd := exec.Command("tar", "czf", tarPath, "-C", localPath, ".")
	tarCmd.Stdout = os.Stdout
	tarCmd.Stderr = os.Stderr
	if err := tarCmd.Run(); err != nil {
		return fmt.Errorf("could not create tarball: %w", err)
	}

	// Upload the tarball as a single file.
	tarName := filepath.Base(tarPath)
	remoteArchive := "/sandbox/" + tarName
	uploadCmd := exec.Command("openshell", "sandbox", "upload", name, tarPath, remoteArchive)
	uploadCmd.Stdout = os.Stdout
	uploadCmd.Stderr = os.Stderr
	if err := uploadCmd.Run(); err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	// Extract the tarball inside the sandbox.
	extractScript := fmt.Sprintf("mkdir -p %s && tar xzf %s -C %s && rm -f %s", remotePath, remoteArchive, remotePath, remoteArchive)
	if _, err := Exec(name, extractScript); err != nil {
		return fmt.Errorf("could not extract tarball in sandbox: %w", err)
	}

	return nil
}

// Download transfers a remote directory from the sandbox to the local
// filesystem using a tarball to preserve dotfiles. The openshell directory
// download may strip dotfiles, so we tar inside the sandbox, download the
// archive, and extract it locally.
func Download(name, remotePath, localPath string) error {
	// Create a tarball inside the sandbox.
	remoteArchive := "/sandbox/od-download.tar.gz"
	tarScript := fmt.Sprintf("tar czf %s -C %s .", remoteArchive, remotePath)
	if _, err := Exec(name, tarScript); err != nil {
		return fmt.Errorf("could not create tarball in sandbox: %w", err)
	}

	// Download the tarball as a single file.
	tmpFile, err := os.CreateTemp("", "od-download-*.tar.gz")
	if err != nil {
		return fmt.Errorf("could not create temp file: %w", err)
	}
	tarPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(tarPath)

	downloadCmd := exec.Command("openshell", "sandbox", "download", name, remoteArchive, tarPath)
	downloadCmd.Stdout = os.Stdout
	downloadCmd.Stderr = os.Stderr
	if err := downloadCmd.Run(); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Extract locally into the target directory.
	if err := os.MkdirAll(localPath, 0o755); err != nil {
		return fmt.Errorf("could not create local directory: %w", err)
	}
	extractCmd := exec.Command("tar", "xzf", tarPath, "-C", localPath)
	extractCmd.Stdout = os.Stdout
	extractCmd.Stderr = os.Stderr
	if err := extractCmd.Run(); err != nil {
		return fmt.Errorf("could not extract tarball locally: %w", err)
	}

	// Clean up remote archive.
	_, _ = Exec(name, fmt.Sprintf("rm -f %s", remoteArchive))

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
