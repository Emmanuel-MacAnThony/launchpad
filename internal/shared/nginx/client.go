package nginx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"text/template"

	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
)

var confTmpl = template.Must(template.New("nginx").Parse(confTemplate))

type Client struct {
	baseDir string
}

func NewClient(baseDir string) *Client {
	return &Client{baseDir: baseDir}
}

func (c *Client) WriteConfig(serviceID string, opts ...func(*Config)) error {
	cfg := &Config{ServiceID: serviceID}
	for _, opt := range opts {
		opt(cfg)
	}

	dir := filepath.Join(c.baseDir, serviceID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating service dir: %w", err)
	}

	jsonBytes, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshalling config: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.json"), jsonBytes, 0644); err != nil {
		return fmt.Errorf("writing config.json: %w", err)
	}

	if err := c.writeConf(dir, cfg, deploydomain.SlotBlue); err != nil {
		return err
	}

	return nil
}

func (c *Client) Switch(serviceID string, slot deploydomain.Slot) error {
	dir := filepath.Join(c.baseDir, serviceID)

	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		return fmt.Errorf("reading config.json: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parsing config.json: %w", err)
	}

	if err := c.writeConf(dir, &cfg, slot); err != nil {
		return err
	}

	return nil
}

func (c *Client) DeleteConfig(serviceID string) error {
	if err := os.RemoveAll(filepath.Join(c.baseDir, serviceID)); err != nil {
		return fmt.Errorf("deleting service dir: %w", err)
	}
	return nil
}

func (c *Client) ReloadNginx() error {
	if err := exec.Command("nginx", "-s", "reload").Run(); err != nil {
		return fmt.Errorf("reloading nginx: %w", err)
	}
	return nil
}

func (c *Client) writeConf(dir string, cfg *Config, activeSlot deploydomain.Slot) error {
	var buf bytes.Buffer
	if err := confTmpl.Execute(&buf, templateData{Config: *cfg, ActiveSlot: string(activeSlot)}); err != nil {
		return fmt.Errorf("rendering nginx template: %w", err)
	}
	confPath := filepath.Join(dir, filepath.Base(dir)+".conf")
	if err := os.WriteFile(confPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("writing nginx conf: %w", err)
	}
	return nil
}
