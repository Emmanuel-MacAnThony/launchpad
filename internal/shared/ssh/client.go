package ssh

import (
	"fmt"
	"os"
	"strings"

	"golang.org/x/crypto/ssh"
)

type SSHClient interface {
	AreFree(ports ...int) (bool, error)
}

type Client struct {
	host    string
	user    string
	keyPath string
}

type Factory struct{}

func (f *Factory) New(host, user, keyPath string) SSHClient {
	return &Client{host: host, user: user, keyPath: keyPath}
}

func NewClient(host, user, keyPath string) *Client {
	return &Client{host: host, user: user, keyPath: keyPath}
}

func (c *Client) AreFree(ports ...int) (bool, error) {
	conn, err := c.dial()
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

func (c *Client) dial() (*ssh.Client, error) {
	keyBytes, err := os.ReadFile(c.keyPath)
	if err != nil {
		return nil, fmt.Errorf("reading key file: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing private key: %w", err)
	}

	cfg := &ssh.ClientConfig{
		User:            c.user,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		// InsecureIgnoreHostKey skips server identity verification (MITM risk).
		// In production, verify against a known host key from config or known_hosts.
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	conn, err := ssh.Dial("tcp", c.host+":22", cfg)
	if err != nil {
		return nil, fmt.Errorf("connecting to %s: %w", c.host, err)
	}

	return conn, nil
}
