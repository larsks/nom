package config

import (
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/guyfedwards/nom/v2/internal/constants"
)

var (
	ErrFeedAlreadyExists = errors.New("config.AddFeed: feed already exists")
)

type Feed struct {
	URL  string `yaml:"url"`
	Name string `yaml:"name,omitempty"`
}

type MinifluxBackend struct {
	Host   string `yaml:"host"`
	APIKey string `yaml:"api_key"`
}

type FreshRSSBackend struct {
	Host       string `yaml:"host"`
	User       string `yaml:"user"`
	Password   string `yaml:"password"`
	PrefixCats bool   `yaml:"prefixCats"`
}

type Backends struct {
	Miniflux *MinifluxBackend `yaml:"miniflux,omitempty"`
	FreshRSS *FreshRSSBackend `yaml:"freshrss,omitempty"`
}

type Opener struct {
	Regex    string `yaml:"regex"`
	Cmd      string `yaml:"cmd"`
	Takeover bool   `yaml:"takeover"`
}

type Theme struct {
	Glamour           string `yaml:"glamour,omitempty"`
	TitleColor        string `yaml:"titleColor,omitempty"`
	TitleColorFg      string `yaml:"titleColorFg,omitempty"`
	FilterColor       string `yaml:"filterColor,omitempty"`
	SelectedItemColor string `yaml:"selectedItemColor,omitempty"`
	ReadIcon          string `yaml:"readIcon,omitempty"`
}

type FilterConfig struct {
	DefaultIncludeFeedName bool `yaml:"defaultIncludeFeedName"`
}

// need to add to Load() below if loading from config file
type Config struct {
	ConfigPath     string
	ShowFavourites bool `yaml:"showfavourites,omitempty"`
	Version        string
	ConfigDir      string       `yaml:"-"`
	Pager          string       `yaml:"pager,omitempty"`
	Feeds          []Feed       `yaml:"feeds"`
	Database       string       `yaml:"database"`
	Ordering       string       `yaml:"ordering"`
	Filtering      FilterConfig `yaml:"filtering"`
	// Preview feeds are distinguished from Feeds because we don't want to inadvertenly write those into the config file.
	PreviewFeeds    []Feed       `yaml:"previewfeeds,omitempty"`
	Backends        *Backends    `yaml:"backends,omitempty"`
	ShowRead        bool         `yaml:"showread,omitempty"`
	AutoRead        bool         `yaml:"autoread,omitempty"`
	Openers         []Opener     `yaml:"openers,omitempty"`
	Theme           Theme        `yaml:"theme,omitempty"`
	HTTPOptions     *HTTPOptions `yaml:"http,omitempty"`
	RefreshInterval int          `yaml:"refreshinterval,omitempty"`
}

var Defaults = Config{
	Database: "nom.db",
	Ordering: constants.DefaultOrdering,
	Theme: Theme{
		Glamour:           "dark",
		SelectedItemColor: "170",
		TitleColor:        "62",
		TitleColorFg:      "231",
		FilterColor:       "62",
		ReadIcon:          "\u2713",
	},
	HTTPOptions: &HTTPOptions{
		MinTLSVersion: tls.VersionName(tls.VersionTLS12),
	},
	Filtering: FilterConfig{
		DefaultIncludeFeedName: false,
	},
	RefreshInterval: 0,
	Feeds:           []Feed{},
}

var sampleConfigFile string = `# See https://github.com/guyfedwards/nom/blob/master/README.md
# for configuration documentation.

feeds:
- name: Nom releases
  url: http://github.com/guyfedwards/nom/releases.atom
- name: Github changelog
  url: https://github.blog/changelog/feed/
`

func (c *Config) ToggleShowRead() {
	c.ShowRead = !c.ShowRead
}

func (c *Config) ToggleShowFavourites() {
	c.ShowFavourites = !c.ShowFavourites
}

func New(configPath string, version string) (*Config, error) {
	var configDir string

	if configPath == "" {
		userConfigDir, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("config.New: %w", err)
		}

		configDir = filepath.Join(userConfigDir, "nom")
		configPath = filepath.Join(configDir, "config.yml")
	} else {
		configDir = filepath.Dir(configPath)
	}

	return &Config{
		ConfigPath: configPath,
		ConfigDir:  configDir,
		Version:    version,
	}, nil
}

func (c *Config) IsPreviewMode() bool {
	return len(c.PreviewFeeds) > 0
}

func (c *Config) Load() error {
	err := c.setupConfigDir()
	if err != nil {
		return fmt.Errorf("config Load: %w", err)
	}

	rawData, err := os.ReadFile(c.ConfigPath)
	if err != nil {
		return fmt.Errorf("config.Load: %w", err)
	}

	// Unmarshal directly into config
	err = yaml.Unmarshal(rawData, c)
	if err != nil {
		return fmt.Errorf("config.Read: %w", err)
	}

	// Apply defaults for zero values
	c.applyDefaults()

	// Validate HTTPOptions
	if c.HTTPOptions != nil {
		if _, err := TLSVersion(c.HTTPOptions.MinTLSVersion); err != nil {
			return err
		}
	}

	// Process backends (requires HTTP calls)
	if c.Backends != nil {
		if c.Backends.Miniflux != nil {
			mffeeds, err := getMinifluxFeeds(c.Backends.Miniflux)
			if err != nil {
				return err
			}
			c.Feeds = append(c.Feeds, mffeeds...)
		}

		if c.Backends.FreshRSS != nil {
			freshfeeds, err := getFreshRSSFeeds(c.Backends.FreshRSS)
			if err != nil {
				return err
			}
			c.Feeds = append(c.Feeds, freshfeeds...)
		}
	}

	return nil
}

// Write writes to a config file
func (c *Config) Write() error {
	str, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("config.Write: %w", err)
	}

	err = os.WriteFile(c.ConfigPath, []byte(str), 0655)
	if err != nil {
		return fmt.Errorf("config.Write: %w", err)
	}

	return nil
}

func (c *Config) AddFeed(feed Feed) error {
	err := c.Load()
	if err != nil {
		return fmt.Errorf("config.AddFeed: %w", err)
	}

	for _, f := range c.Feeds {
		if f.URL == feed.URL {
			return ErrFeedAlreadyExists
		}
	}

	c.Feeds = append(c.Feeds, feed)

	err = c.Write()
	if err != nil {
		return fmt.Errorf("config.AddFeed: %w", err)
	}

	return nil
}

func (c *Config) GetFeeds() []Feed {
	if c.IsPreviewMode() {
		return c.PreviewFeeds
	}

	return c.Feeds
}

func (c *Config) setupConfigDir() error {
	_, err := os.Stat(c.ConfigPath)

	// if configFile exists, do nothing
	if !errors.Is(err, os.ErrNotExist) {
		return nil
	}

	// if not, create directory. noop if directory exists
	err = os.MkdirAll(c.ConfigDir, 0755)
	if err != nil {
		return fmt.Errorf("setupConfigDir: %w", err)
	}

	// then create the file
	err = os.WriteFile(c.ConfigPath, []byte(sampleConfigFile), 0644)
	if err != nil {
		return fmt.Errorf("setupConfigDir: %w", err)
	}

	return err
}

// applyDefaults fills in zero values from Defaults
func (c *Config) applyDefaults() {
	if c.Database == "" {
		c.Database = Defaults.Database
	}

	if c.Ordering == "" {
		c.Ordering = Defaults.Ordering
	}

	// Apply theme defaults for zero values
	if c.Theme.Glamour == "" {
		c.Theme.Glamour = Defaults.Theme.Glamour
	}
	if c.Theme.SelectedItemColor == "" {
		c.Theme.SelectedItemColor = Defaults.Theme.SelectedItemColor
	}
	if c.Theme.TitleColor == "" {
		c.Theme.TitleColor = Defaults.Theme.TitleColor
	}
	if c.Theme.TitleColorFg == "" {
		c.Theme.TitleColorFg = Defaults.Theme.TitleColorFg
	}
	if c.Theme.FilterColor == "" {
		c.Theme.FilterColor = Defaults.Theme.FilterColor
	}
	if c.Theme.ReadIcon == "" {
		c.Theme.ReadIcon = Defaults.Theme.ReadIcon
	}

	if c.HTTPOptions == nil {
		c.HTTPOptions = Defaults.HTTPOptions
	} else if c.HTTPOptions.MinTLSVersion == "" {
		c.HTTPOptions.MinTLSVersion = Defaults.HTTPOptions.MinTLSVersion
	}
}

// ApplyCLIOverrides applies CLI flags (highest priority)
func (c *Config) ApplyCLIOverrides(pager string, previewFeeds []string) {
	if pager != "" {
		c.Pager = pager
	}

	if len(previewFeeds) > 0 {
		c.PreviewFeeds = make([]Feed, len(previewFeeds))
		for i, url := range previewFeeds {
			c.PreviewFeeds[i] = Feed{URL: url}
		}
	}
}
