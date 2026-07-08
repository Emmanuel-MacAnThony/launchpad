package ssh

// Client performs SSH operations on a remote host.
// Construct with the host's connection details; methods dial on each call.
// Real implementation uses golang.org/x/crypto/ssh.
type Client struct {
	Host    string
	User    string
	KeyPath string
}

func NewClient(host, user, keyPath string) *Client {
	return &Client{Host: host, User: user, KeyPath: keyPath}
}

// FindFreePorts scans the host and returns `count` available port numbers.
func (c *Client) FindFreePorts(count int) ([]int, error) {
	panic("ssh.Client.FindFreePorts: not implemented")
}

// AreFree reports whether all given ports are unoccupied on the host.
// Returns false (not an error) when a port is in use.
// Returns an error only when the SSH connection or remote command fails.
func (c *Client) AreFree(ports ...int) (bool, error) {
	panic("ssh.Client.AreFree: not implemented")
}
