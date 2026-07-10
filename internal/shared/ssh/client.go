package ssh

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

// SSHConfig holds the connection parameters for an SSH session.
type SSHConfig struct {
	Host    string
	User    string
	KeyPath string
}

// SSHResult holds the output of a remote command.
type SSHResult struct {
	Stdout string
	Stderr string
}

// SSHClient is the interface satisfied by Client (dial-per-call, used by create use case).
type SSHClient interface {
	AreFree(ports ...int) (bool, error)
}

// Client opens a new SSH connection for each operation.
// Used by the create use case to check port availability.
type Client struct {
	host    string
	user    string
	keyPath string
}

// Factory creates SSH clients and executors.
type Factory struct{}

func (f *Factory) New(host, user, keyPath string) SSHClient {
	return &Client{host: host, user: user, keyPath: keyPath}
}

// NewExecutor dials once and returns a persistent Executor for the caller's lifetime.
// The caller must call Close() when done.
func (f *Factory) NewExecutor(cfg SSHConfig) (*Executor, error) {
	conn, err := dial(cfg.Host, cfg.User, cfg.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("dialing %s: %w", cfg.Host, err)
	}
	return &Executor{conn: conn}, nil
}

func NewClient(host, user, keyPath string) *Client {
	return &Client{host: host, user: user, keyPath: keyPath}
}

func (c *Client) AreFree(ports ...int) (bool, error) {
	conn, err := dial(c.host, c.user, c.keyPath)
	if err != nil {
		return false, fmt.Errorf("dialing ssh: %w", err)
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return false, fmt.Errorf("opening ssh session: %w", err)
	}
	defer session.Close()

	out, err := session.Output("ss -tln")
	if err != nil {
		return false, fmt.Errorf("running ss: %w", err)
	}

	output := string(out)
	for _, port := range ports {
		if strings.Contains(output, fmt.Sprintf(":%d ", port)) {
			return false, nil
		}
	}

	return true, nil
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

// dial is shared by both Client and Factory.NewExecutor.
func dial(host, user, keyPath string) (*ssh.Client, error) {
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("reading key file: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}

	cfg := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		// InsecureIgnoreHostKey skips server identity verification (MITM risk).
		// In production, verify against a known host key from config or known_hosts.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", host+":22", cfg)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", host, err)
	}

	return conn, nil
}
