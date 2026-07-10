package nginx_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/shared/nginx"
)

func TestWriteConfig_CreatesDirectory(t *testing.T) {
	baseDir := t.TempDir()
	c := nginx.NewClient(baseDir)

	if err := c.WriteConfig("svc-1", withDomain("app.example.com"), withHost("10.0.0.1"), withBluePort(3001), withGreenPort(3002)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := os.Stat(filepath.Join(baseDir, "svc-1"))
	if err != nil {
		t.Fatalf("expected directory to exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected a directory, got a file")
	}
}

func TestWriteConfig_WritesConfigJSON(t *testing.T) {
	baseDir := t.TempDir()
	c := nginx.NewClient(baseDir)

	if err := c.WriteConfig("svc-1", withDomain("app.example.com"), withHost("10.0.0.1"), withBluePort(3001), withGreenPort(3002)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(baseDir, "svc-1", "config.json"))
	if err != nil {
		t.Fatalf("expected config.json to exist: %v", err)
	}

	var cfg nginx.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("failed to parse config.json: %v", err)
	}

	if cfg.ServiceID != "svc-1" {
		t.Errorf("expected service_id=svc-1, got %s", cfg.ServiceID)
	}
	if cfg.Domain != "app.example.com" {
		t.Errorf("expected domain=app.example.com, got %s", cfg.Domain)
	}
	if cfg.Host != "10.0.0.1" {
		t.Errorf("expected host=10.0.0.1, got %s", cfg.Host)
	}
	if cfg.BluePort != 3001 {
		t.Errorf("expected blue_port=3001, got %d", cfg.BluePort)
	}
	if cfg.GreenPort != 3002 {
		t.Errorf("expected green_port=3002, got %d", cfg.GreenPort)
	}
}

func TestWriteConfig_WritesNginxConf(t *testing.T) {
	baseDir := t.TempDir()
	c := nginx.NewClient(baseDir)

	if err := c.WriteConfig("svc-1", withDomain("app.example.com"), withHost("10.0.0.1"), withBluePort(3001), withGreenPort(3002)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(baseDir, "svc-1", "svc-1.conf"))
	if err != nil {
		t.Fatalf("expected svc-1.conf to exist: %v", err)
	}

	conf := string(data)

	assertContains(t, conf, "server_name app.example.com")
	assertContains(t, conf, "server 10.0.0.1:3001")
	assertContains(t, conf, "server 10.0.0.1:3002")
	assertContains(t, conf, "proxy_pass http://svc-1_blue")
	if strings.Contains(conf, "proxy_pass http://svc-1_green") {
		t.Error("expected default proxy_pass to be blue, not green")
	}
}

func TestSwitch_UpdatesProxyPass(t *testing.T) {
	baseDir := t.TempDir()
	c := nginx.NewClient(baseDir)

	if err := c.WriteConfig("svc-1", withDomain("app.example.com"), withHost("10.0.0.1"), withBluePort(3001), withGreenPort(3002)); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	if err := c.Switch("svc-1", deploydomain.SlotGreen); err != nil {
		t.Fatalf("Switch: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(baseDir, "svc-1", "svc-1.conf"))
	if err != nil {
		t.Fatalf("reading conf: %v", err)
	}
	conf := string(data)

	assertContains(t, conf, "proxy_pass http://svc-1_green")
	if strings.Contains(conf, "proxy_pass http://svc-1_blue") {
		t.Error("expected proxy_pass to switch to green, still shows blue")
	}
}

func TestSwitch_DoesNotModifyConfigJSON(t *testing.T) {
	baseDir := t.TempDir()
	c := nginx.NewClient(baseDir)

	if err := c.WriteConfig("svc-1", withDomain("app.example.com"), withHost("10.0.0.1"), withBluePort(3001), withGreenPort(3002)); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	before, _ := os.ReadFile(filepath.Join(baseDir, "svc-1", "config.json"))

	if err := c.Switch("svc-1", deploydomain.SlotGreen); err != nil {
		t.Fatalf("Switch: %v", err)
	}

	after, _ := os.ReadFile(filepath.Join(baseDir, "svc-1", "config.json"))
	if string(before) != string(after) {
		t.Error("expected config.json to be unchanged after Switch")
	}
}

func TestSwitch_ErrorWhenServiceNotFound(t *testing.T) {
	baseDir := t.TempDir()
	c := nginx.NewClient(baseDir)

	if err := c.Switch("nonexistent", deploydomain.SlotGreen); err == nil {
		t.Fatal("expected error for missing service, got nil")
	}
}

func TestDeleteConfig_RemovesServiceDirectory(t *testing.T) {
	baseDir := t.TempDir()
	c := nginx.NewClient(baseDir)

	if err := c.WriteConfig("svc-1", withDomain("app.example.com"), withHost("10.0.0.1"), withBluePort(3001), withGreenPort(3002)); err != nil {
		t.Fatalf("WriteConfig: %v", err)
	}

	if err := c.DeleteConfig("svc-1"); err != nil {
		t.Fatalf("DeleteConfig: %v", err)
	}

	if _, err := os.Stat(filepath.Join(baseDir, "svc-1")); !os.IsNotExist(err) {
		t.Error("expected service directory to be removed")
	}
}

func TestDeleteConfig_NoErrorOnNonExistentService(t *testing.T) {
	baseDir := t.TempDir()
	c := nginx.NewClient(baseDir)

	if err := c.DeleteConfig("nonexistent"); err != nil {
		t.Fatalf("expected no error for nonexistent service, got %v", err)
	}
}

// --- helpers ---

func assertContains(t *testing.T, content, substr string) {
	t.Helper()
	if !strings.Contains(content, substr) {
		t.Errorf("expected conf to contain %q", substr)
	}
}

func withDomain(d string) func(*nginx.Config) {
	return func(c *nginx.Config) { c.Domain = d }
}

func withHost(h string) func(*nginx.Config) {
	return func(c *nginx.Config) { c.Host = h }
}

func withBluePort(p int) func(*nginx.Config) {
	return func(c *nginx.Config) { c.BluePort = p }
}

func withGreenPort(p int) func(*nginx.Config) {
	return func(c *nginx.Config) { c.GreenPort = p }
}
