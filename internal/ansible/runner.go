package ansible

import (
	"bytes"
	"context"
	"fmt"
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
	playbookDir  string
	sshKeyPath   string
	elbenchoDeb  string
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

func (r *Runner) ProvisionWorkers(ctx context.Context, targets []WorkerTarget) error {
	inv, err := r.GenerateInventory(targets)
	if err != nil {
		return err
	}
	invFile := filepath.Join(os.TempDir(), "benchere_inventory.ini")
	if err := os.WriteFile(invFile, []byte(inv), 0600); err != nil {
		return err
	}
	defer os.Remove(invFile)

	cmd := exec.CommandContext(ctx,
		"ansible-playbook",
		"-i", invFile,
		filepath.Join(r.playbookDir, "provision_worker.yml"),
		"--extra-vars", fmt.Sprintf("elbencho_deb_local=%s", r.elbenchoDeb),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
