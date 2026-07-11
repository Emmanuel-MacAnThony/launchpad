package ssh

import (
	"bytes"
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
)

// SSHConfig holds the connection parameters for an SSH session.
type SSHConfig struct {
	Host     string
	User     string
	KeyBytes []byte // decrypted private key content provided by the customer
}

// SSHResult holds the output of a remote command.
type SSHResult struct {
	Stdout string
	Stderr string
}

// Factory creates SSH executors.
type Factory struct{}

// NewExecutor dials once and returns a persistent Executor for the caller's lifetime.
// The caller must call Close() when done.
func (f *Factory) NewExecutor(cfg SSHConfig) (*Executor, error) {
	conn, err := dial(cfg.Host, cfg.User, cfg.KeyBytes)
	if err != nil {
		return nil, fmt.Errorf("dialing %s: %w", cfg.Host, err)
	}
	return &Executor{conn: conn}, nil
}

// Executor holds an open SSH connection for a worker's lifetime.
// Create via Factory.NewExecutor; close with Close() when done.
type Executor struct {
	conn *ssh.Client
}

func (e *Executor) Run(cmd string) (SSHResult, error) {
	session, err := e.conn.NewSession()
	if err != nil {
		return SSHResult{}, fmt.Errorf("opening session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(cmd); err != nil {
		return SSHResult{Stdout: stdout.String(), Stderr: stderr.String()},
			fmt.Errorf("running %q: %w (stderr: %s)", cmd, err, stderr.String())
	}

	return SSHResult{Stdout: stdout.String(), Stderr: stderr.String()}, nil
}

func (e *Executor) Upload(localPath, remotePath string) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("reading local file: %w", err)
	}

	session, err := e.conn.NewSession()
	if err != nil {
		return fmt.Errorf("opening session: %w", err)
	}
	defer session.Close()

	session.Stdin = bytes.NewReader(data)
	if err := session.Run(fmt.Sprintf("cat > %s", remotePath)); err != nil {
		return fmt.Errorf("uploading to %s: %w", remotePath, err)
	}

	return nil
}

func (e *Executor) Close() error {
	return e.conn.Close()
}

func dial(host, user string, keyBytes []byte) (*ssh.Client, error) {
	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}

	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", host+":22", cfg)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", host, err)
	}

	return conn, nil
}
