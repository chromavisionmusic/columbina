package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const DefaultEmbedColor = 0x9c3f73

type HexColor int

func (c *HexColor) UnmarshalYAML(value *yaml.Node) error {
	var i int
	if value.Tag == "!!int" {
		if err := value.Decode(&i); err == nil {
			*c = HexColor(i)
			return nil
		}
	}

	var s string
	if err := value.Decode(&s); err == nil {
		s = strings.TrimSpace(s)
		s = strings.TrimPrefix(s, "#")
		s = strings.TrimPrefix(s, "0x")
		s = strings.TrimPrefix(s, "0X")
		if s == "" {
			*c = DefaultEmbedColor
			return nil
		}
		val, err := strconv.ParseInt(s, 16, 32)
		if err != nil {
			return fmt.Errorf("invalid hex color %q: %w", value.Value, err)
		}
		*c = HexColor(val)
		return nil
	}

	if err := value.Decode(&i); err == nil {
		*c = HexColor(i)
		return nil
	}

	return fmt.Errorf("cannot parse color into hex color: %s", value.Value)
}

func (c HexColor) Int() int {
	return int(c)
}

func (c HexColor) String() string {
	return fmt.Sprintf("#%06x", int(c))
}

type Config struct {
	Bot BotConfig `yaml:"bot"`
}

func (c *Config) ColorInt() int {
	if c == nil {
		return DefaultEmbedColor
	}
	return c.Bot.ColorInt()
}

type User struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name"`
}

type Links struct {
	Invite  string `yaml:"invite"`
	Website string `yaml:"website"`
	Server  string `yaml:"server"`
	Github  string `yaml:"github"`
}

type BotConfig struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	Version      string   `yaml:"version"`
	Color        HexColor `yaml:"color"`
	Owner        User     `yaml:"owner"`
	Designer     User     `yaml:"designer"`
	Contributors []User   `yaml:"contributors,omitempty"`
	Links        Links    `yaml:"links"`
	Presence     Presence `yaml:"presence"`
}

func (b BotConfig) ColorInt() int {
	if b.Color == 0 {
		return DefaultEmbedColor
	}
	return b.Color.Int()
}

type Presence struct {
	Status       string `yaml:"status"`
	ActivityType string `yaml:"activity_type"`
	Activity     string `yaml:"activity"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	cfg := Config{
		Bot: BotConfig{
			Color: DefaultEmbedColor,
		},
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	if cfg.Bot.Name == "" {
		return nil, fmt.Errorf("bot name is required")
	}

	if cfg.Bot.Version == "" {
		return nil, fmt.Errorf("bot version is required")
	}

	return &cfg, nil
}
