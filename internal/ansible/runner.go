package ansible

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"
)

var inventoryTmpl = template.Must(template.New("inv").Parse(
	"[workers]\n{{range .Targets}}{{.IP}} ansible_user=root ansible_ssh_private_key_file={{$.SSHKeyPath}} ansible_ssh_common_args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null'\n{{end}}",
))

type WorkerTarget struct {
	IP string
}

type Runner struct {
	playbookDir string
	sshKeyPath  string
	elbenchoDeb string
}

func NewRunner(playbookDir, sshKeyPath, elbenchoDeb string) *Runner {
	return &Runner{playbookDir: playbookDir, sshKeyPath: sshKeyPath, elbenchoDeb: elbenchoDeb}
}

func (r *Runner) GenerateInventory(targets []WorkerTarget) (string, error) {
	var buf bytes.Buffer
	data := struct {
		Targets    []WorkerTarget
		SSHKeyPath string
	}{Targets: targets, SSHKeyPath: r.sshKeyPath}
	if err := inventoryTmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ProvisionWorkers runs the provision_worker playbook against the given
// targets. When outputDir is non-empty, the literal command line plus the
// playbook stdout/stderr are persisted to outputDir/provision.{cmd,stdout,stderr}
// alongside a copy of the generated inventory, so a debug bundle can ship them.
// Failures to persist are logged but do not block the run.
func (r *Runner) ProvisionWorkers(ctx context.Context, targets []WorkerTarget, outputDir string) error {
	inv, err := r.GenerateInventory(targets)
	if err != nil {
		return err
	}
	invFile := filepath.Join(os.TempDir(), "benchere_inventory.ini")
	if err := os.WriteFile(invFile, []byte(inv), 0600); err != nil {
		return err
	}
	defer os.Remove(invFile)

	playbook := filepath.Join(r.playbookDir, "provision_worker.yml")
	args := []string{
		"-i", invFile,
		playbook,
		"--extra-vars", fmt.Sprintf("elbencho_deb_local=%s", r.elbenchoDeb),
	}
	cmd := exec.CommandContext(ctx, "ansible-playbook", args...)

	var stdoutW io.Writer = os.Stdout
	var stderrW io.Writer = os.Stderr
	var stdoutFile, stderrFile *os.File

	if outputDir != "" {
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "[ansible] mkdir %s: %v\n", outputDir, err)
		} else {
			cmdLine := "ansible-playbook " + joinArgsForLog(args) + "\n"
			_ = os.WriteFile(filepath.Join(outputDir, "provision.cmd"), []byte(cmdLine), 0o644)
			_ = os.WriteFile(filepath.Join(outputDir, "inventory.ini"), []byte(inv), 0o600)
			if f, err := os.Create(filepath.Join(outputDir, "provision.stdout")); err == nil {
				stdoutFile = f
				stdoutW = io.MultiWriter(os.Stdout, f)
			}
			if f, err := os.Create(filepath.Join(outputDir, "provision.stderr")); err == nil {
				stderrFile = f
				stderrW = io.MultiWriter(os.Stderr, f)
			}
		}
	}

	cmd.Stdout = stdoutW
	cmd.Stderr = stderrW
	runErr := cmd.Run()

	if stdoutFile != nil {
		stdoutFile.Close()
	}
	if stderrFile != nil {
		stderrFile.Close()
	}
	return runErr
}

// joinArgsForLog wraps any arg containing whitespace in double quotes so the
// captured command line is safe to copy into a shell.
func joinArgsForLog(args []string) string {
	var buf bytes.Buffer
	for i, a := range args {
		if i > 0 {
			buf.WriteByte(' ')
		}
		needsQuote := false
		for _, c := range a {
			if c == ' ' || c == '\t' {
				needsQuote = true
				break
			}
		}
		if needsQuote {
			buf.WriteByte('"')
			for _, c := range a {
				if c == '"' {
					buf.WriteByte('\\')
				}
				buf.WriteRune(c)
			}
			buf.WriteByte('"')
		} else {
			buf.WriteString(a)
		}
	}
	return buf.String()
}
