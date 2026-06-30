package install

import (
	"reflect"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/aliyun/aliyun_assist_client/agent/util/systemdutil"
)

func TestServiceConfig(t *testing.T) {
	t.Run("systemd environment", func(t *testing.T) {
		defer gomonkey.ApplyFunc(systemdutil.IsRunningSystemd, func() bool {
			return true
		}).Reset()

		cfg := ServiceConfig()
		if cfg.Name != "aliyun" {
			t.Errorf("expected Name='aliyun', got %q", cfg.Name)
		}
		if cfg.DisplayName != "Aliyun Assist Service" {
			t.Errorf("unexpected DisplayName: %q", cfg.DisplayName)
		}
		if cfg.Description != "Aliyun Assist" {
			t.Errorf("unexpected Description: %q", cfg.Description)
		}
		expectedDepends := []string{"After=network-online.target", "Wants=network-online.target"}
		if !reflect.DeepEqual(cfg.Dependencies, expectedDepends) {
			t.Errorf("Dependencies = %v, want %v", cfg.Dependencies, expectedDepends)
		}
		if logOutput, ok := cfg.Option["LogOutput"].(bool); !ok || logOutput != false {
			t.Errorf("LogOutput = %v (type %T), want false", cfg.Option["LogOutput"], cfg.Option["LogOutput"])
		}
		if restart, ok := cfg.Option["Restart"].(string); !ok || restart != "on-failure" {
			t.Errorf("Restart = %v (type %T), want 'on-failure'", cfg.Option["Restart"], cfg.Option["Restart"])
		}
		for _, key := range []string{"SystemdScript", "SysvScript", "UpstartScript"} {
			if _, exists := cfg.Option[key]; !exists {
				t.Errorf("Option %q is missing", key)
			}
		}
	})

	t.Run("non-systemd environment", func(t *testing.T) {
		defer gomonkey.ApplyFunc(systemdutil.IsRunningSystemd, func() bool {
			return false
		}).Reset()

		cfg := ServiceConfig()
		if cfg.Name != "aliyun-service" {
			t.Errorf("expected Name='aliyun-service', got %q", cfg.Name)
		}
		if cfg.DisplayName != "Aliyun Assist Service" {
			t.Errorf("unexpected DisplayName: %q", cfg.DisplayName)
		}
		if cfg.Description != "Aliyun Assist" {
			t.Errorf("unexpected Description: %q", cfg.Description)
		}

		if len(cfg.Dependencies) != 0 {
			t.Errorf("expected no Dependencies, got %v", cfg.Dependencies)
		}
		if logOutput, ok := cfg.Option["LogOutput"].(bool); !ok || logOutput != true {
			t.Errorf("LogOutput = %v (type %T), want true", cfg.Option["LogOutput"], cfg.Option["LogOutput"])
		}
		if _, exists := cfg.Option["Restart"]; exists {
			t.Errorf("Restart should not be set in non-systemd mode, but got %v", cfg.Option["Restart"])
		}
		for _, key := range []string{"SystemdScript", "SysvScript", "UpstartScript"} {
			if _, exists := cfg.Option[key]; !exists {
				t.Errorf("Option %q is missing", key)
			}
		}
	})
}
