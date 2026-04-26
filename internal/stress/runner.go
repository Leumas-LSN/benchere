package stress

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type Config struct {
	Workers   int
	Timeout   int // seconds
	Stressors []string // e.g. ["cpu", "vm", "io"]
}

type Runner struct {
	sshKeyPath string
}

func NewRunner(sshKeyPath string) *Runner {
	return &Runner{sshKeyPath: sshKeyPath}
}

func BuildCommand(cfg Config) string {
	parts := []string{"stress-ng"}
	for _, s := range cfg.Stressors {
		parts = append(parts, fmt.Sprintf("--%s %d", s, cfg.Workers))
	}
	parts = append(parts, fmt.Sprintf("--timeout %ds", cfg.Timeout))
	parts = append(parts, "--metrics-brief")
	return strings.Join(parts, " ")
}

func (r *Runner) sshClient(ip string) (*ssh.Client, error) {
	key, err := os.ReadFile(r.sshKeyPath)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}
	cfg := &ssh.ClientConfig{
		User:            "root",
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	return ssh.Dial("tcp", net.JoinHostPort(ip, "22"), cfg)
}

func (r *Runner) StartOnWorkers(ctx context.Context, ips []string, cfg Config) error {
	cmd := BuildCommand(cfg) + " &"
	for _, ip := range ips {
		client, err := r.sshClient(ip)
		if err != nil {
			return fmt.Errorf("ssh %s: %w", ip, err)
		}
		sess, err := client.NewSession()
		if err != nil {
			client.Close()
			return err
		}
		if err := sess.Start(cmd); err != nil {
			sess.Close()
			client.Close()
			return fmt.Errorf("start stress-ng on %s: %w", ip, err)
		}
		sess.Close()
		client.Close()
	}
	return nil
}

func (r *Runner) StopOnWorkers(_ context.Context, ips []string) error {
	for _, ip := range ips {
		client, err := r.sshClient(ip)
		if err != nil {
			continue
		}
		sess, _ := client.NewSession()
		if sess != nil {
			_ = sess.Run("pkill -f stress-ng")
			sess.Close()
		}
		client.Close()
	}
	return nil
}
