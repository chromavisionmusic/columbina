package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestHexColor_UnmarshalYAML(t *testing.T) {
	tests := []struct {
		name      string
		yamlData  string
		wantColor int
		wantErr   bool
	}{
		{
			name:      "integer value",
			yamlData:  "color: 10239859\n",
			wantColor: 10239859,
			wantErr:   false,
		},
		{
			name:      "hex with hash prefix",
			yamlData:  "color: \"#9c3f73\"\n",
			wantColor: 0x9c3f73,
			wantErr:   false,
		},
		{
			name:      "hex with 0x prefix",
			yamlData:  "color: \"0x123456\"\n",
			wantColor: 0x123456,
			wantErr:   false,
		},
		{
			name:      "hex with 0X prefix uppercase",
			yamlData:  "color: \"0XABCDEF\"\n",
			wantColor: 0xabcdef,
			wantErr:   false,
		},
		{
			name:      "hex plain string",
			yamlData:  "color: \"00ff00\"\n",
			wantColor: 0x00ff00,
			wantErr:   false,
		},
		{
			name:      "empty string defaults to DefaultEmbedColor",
			yamlData:  "color: \"\"\n",
			wantColor: DefaultEmbedColor,
			wantErr:   false,
		},
		{
			name:      "invalid hex string",
			yamlData:  "color: \"xyz123\"\n",
			wantColor: 0,
			wantErr:   true,
		},
		{
			name:      "invalid yaml node type (mapping)",
			yamlData:  "color: {a: 1}\n",
			wantColor: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wrapper struct {
				Color HexColor `yaml:"color"`
			}
			err := yaml.Unmarshal([]byte(tt.yamlData), &wrapper)
			if (err != nil) != tt.wantErr {
				t.Fatalf("yaml.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && wrapper.Color.Int() != tt.wantColor {
				t.Errorf("got color = %x, want %x", wrapper.Color.Int(), tt.wantColor)
			}
		})
	}
}

func TestHexColor_StringAndInt(t *testing.T) {
	c := HexColor(0x1a2b3c)
	if c.Int() != 0x1a2b3c {
		t.Errorf("Int() = %x, want %x", c.Int(), 0x1a2b3c)
	}
	if c.String() != "#1a2b3c" {
		t.Errorf("String() = %q, want %q", c.String(), "#1a2b3c")
	}

	cZero := HexColor(0)
	if cZero.String() != "#000000" {
		t.Errorf("String() = %q, want %q", cZero.String(), "#000000")
	}
}

func TestConfig_ColorInt(t *testing.T) {
	var nilCfg *Config
	if nilCfg.ColorInt() != DefaultEmbedColor {
		t.Errorf("nil Config.ColorInt() = %x, want default %x", nilCfg.ColorInt(), DefaultEmbedColor)
	}

	cfgWithZeroColor := &Config{
		Bot: BotConfig{
			Color: 0,
		},
	}
	if cfgWithZeroColor.ColorInt() != DefaultEmbedColor {
		t.Errorf("zero color ColorInt() = %x, want default %x", cfgWithZeroColor.ColorInt(), DefaultEmbedColor)
	}

	customColor := HexColor(0xff5500)
	cfgWithCustom := &Config{
		Bot: BotConfig{
			Color: customColor,
		},
	}
	if cfgWithCustom.ColorInt() != 0xff5500 {
		t.Errorf("custom color ColorInt() = %x, want %x", cfgWithCustom.ColorInt(), 0xff5500)
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.yml")

	yamlContent := `
bot:
  name: "Columbina"
  description: "A friendly bot"
  version: "1.0.0"
  color: "#336699"
  owner:
    id: "12345"
    name: "OwnerName"
  designer:
    id: "67890"
    name: "DesignerName"
  contributors:
    - id: "11111"
      name: "Contrib1"
  links:
    invite: "https://discord.gg/invite"
    website: "https://example.com"
    server: "https://discord.gg/server"
    github: "https://github.com/repo"
  presence:
    status: "idle"
    activity_type: "playing"
    activity: "with shadows"
`
	if err := os.WriteFile(configFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("failed to write test config file: %v", err)
	}

	cfg, err := Load(configFile)
	if err != nil {
		t.Fatalf("Load() unexpected error: %v", err)
	}

	if cfg.Bot.Name != "Columbina" {
		t.Errorf("Name = %q, want %q", cfg.Bot.Name, "Columbina")
	}
	if cfg.Bot.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", cfg.Bot.Version, "1.0.0")
	}
	if cfg.Bot.Color.Int() != 0x336699 {
		t.Errorf("Color = %x, want %x", cfg.Bot.Color.Int(), 0x336699)
	}
	if cfg.Bot.Owner.Name != "OwnerName" || cfg.Bot.Owner.ID != "12345" {
		t.Errorf("Owner mismatch: %+v", cfg.Bot.Owner)
	}
	if cfg.Bot.Designer.Name != "DesignerName" || cfg.Bot.Designer.ID != "67890" {
		t.Errorf("Designer mismatch: %+v", cfg.Bot.Designer)
	}
	if len(cfg.Bot.Contributors) != 1 || cfg.Bot.Contributors[0].Name != "Contrib1" {
		t.Errorf("Contributors mismatch: %+v", cfg.Bot.Contributors)
	}
	if cfg.Bot.Links.Invite != "https://discord.gg/invite" {
		t.Errorf("Links.Invite = %q", cfg.Bot.Links.Invite)
	}
	if cfg.Bot.Presence.Status != "idle" || cfg.Bot.Presence.Activity != "with shadows" {
		t.Errorf("Presence mismatch: %+v", cfg.Bot.Presence)
	}
}

func TestLoad_Errors(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("non-existent file", func(t *testing.T) {
		_, err := Load(filepath.Join(tempDir, "does-not-exist.yml"))
		if err == nil {
			t.Fatal("expected error for non-existent file, got nil")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		badFile := filepath.Join(tempDir, "bad.yml")
		if err := os.WriteFile(badFile, []byte("bot: [unclosed"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := Load(badFile)
		if err == nil {
			t.Fatal("expected error for bad yaml, got nil")
		}
	})

	t.Run("missing bot name", func(t *testing.T) {
		noNameFile := filepath.Join(tempDir, "no_name.yml")
		if err := os.WriteFile(noNameFile, []byte("bot:\n  version: '1.0.0'\n"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := Load(noNameFile)
		if err == nil || err.Error() != "bot name is required" {
			t.Fatalf("expected 'bot name is required', got %v", err)
		}
	})

	t.Run("missing bot version", func(t *testing.T) {
		noVersionFile := filepath.Join(tempDir, "no_version.yml")
		if err := os.WriteFile(noVersionFile, []byte("bot:\n  name: 'Columbina'\n"), 0644); err != nil {
			t.Fatal(err)
		}
		_, err := Load(noVersionFile)
		if err == nil || err.Error() != "bot version is required" {
			t.Fatalf("expected 'bot version is required', got %v", err)
		}
	})
}
