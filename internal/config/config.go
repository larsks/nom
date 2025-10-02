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
	DefaultConfigDirName = "nom"
	DefaultProfileName   = "nom"
	DefaultCacheDirName  = "nom"
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
	ConfigPath     string       `yaml:"-"`
	Profile        string       `yaml:"-"`
	Create         bool         `yaml:"-"`
	ShowFavourites bool         `yaml:"showfavourites,omitempty"`
	Version        string       `yaml:"-"`
	ConfigDir      string       `yaml:"-"`
	CacheDir       string       `yaml:"-"`
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
	AutoReadBrowser bool         `yaml:"autoreadbrowser,omitempty"`
	Openers         []Opener     `yaml:"openers,omitempty"`
	Theme           Theme        `yaml:"theme,omitempty"`
	HTTPOptions     *HTTPOptions `yaml:"http,omitempty"`
	RefreshInterval int          `yaml:"refreshinterval,omitempty"`
}

var DefaultTheme = Theme{
	Glamour:           "dark",
	SelectedItemColor: "170",
	TitleColor:        "62",
	TitleColorFg:      "231",
	FilterColor:       "62",
	ReadIcon:          "\u2713",
}

func (c *Config) ToggleShowRead() {
	c.ShowRead = !c.ShowRead
}

func (c *Config) ToggleShowFavourites() {
	c.ShowFavourites = !c.ShowFavourites
}

func New() *Config {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		userConfigDir = ""
	}

	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		userCacheDir = ""
	}

	configDir := filepath.Join(userConfigDir, DefaultConfigDirName)
	configPath := filepath.Join(configDir, DefaultProfileName+".yml")
	cacheDir := filepath.Join(userCacheDir, DefaultCacheDirName)

	return &Config{
		ConfigPath:      configPath,
		Profile:         DefaultProfileName,
		ConfigDir:       configDir,
		CacheDir:        cacheDir,
		Database:        DefaultProfileName,
		Pager:           "",
		Feeds:           []Feed{},
		PreviewFeeds:    []Feed{},
		Theme:           DefaultTheme,
		RefreshInterval: 0,
		Ordering:        constants.DefaultOrdering,
		Filtering: FilterConfig{
			DefaultIncludeFeedName: false,
		},
		HTTPOptions: &HTTPOptions{
			MinTLSVersion: tls.VersionName(tls.VersionTLS12),
		},
	}
}

func (c *Config) WithConfigPath(configDir string) *Config {
	if configDir != "" {
		// Ensure ConfigDir ends with separator for consistency
		if configDir[len(configDir)-1] != filepath.Separator {
			configDir = configDir + string(filepath.Separator)
		}
		c.ConfigDir = configDir
		c.ConfigPath = filepath.Join(configDir, c.Profile+".yml")
	}
	return c
}

func (c *Config) WithProfile(profile string) *Config {
	if profile != "" {
		c.Profile = profile
		c.ConfigPath = filepath.Join(c.ConfigDir, profile+".yml")
		c.Database = profile
	}
	return c
}

func (c *Config) WithPager(pager string) *Config {
	if pager != "" {
		c.Pager = pager
	}
	return c
}

func (c *Config) WithPreviewFeeds(previewFeeds []string) *Config {
	if len(previewFeeds) > 0 {
		var f []Feed
		for _, feedURL := range previewFeeds {
			f = append(f, Feed{URL: feedURL})
		}
		c.PreviewFeeds = f
	}
	return c
}

func (c *Config) WithVersion(version string) *Config {
	c.Version = version
	return c
}

func (c *Config) WithDatabase(database string) *Config {
	if database != "" {
		c.Database = database
	}
	return c
}

func (c *Config) WithDataDir(dataDir string) *Config {
	if dataDir != "" {
		c.CacheDir = dataDir
	}
	return c
}

func (c *Config) WithCreate(create bool) *Config {
	c.Create = create
	return c
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

	// manually set config values from fileconfig, messy solve for config priority
	var fileConfig Config
	err = yaml.Unmarshal(rawData, &fileConfig)
	if err != nil {
		return fmt.Errorf("config.Read: %w", err)
	}

	c.ShowRead = fileConfig.ShowRead
	c.AutoRead = fileConfig.AutoRead
	c.AutoReadBrowser = fileConfig.AutoReadBrowser
	c.Feeds = fileConfig.Feeds
	if fileConfig.Database != "" {
		c.Database = fileConfig.Database
	}
	c.Openers = fileConfig.Openers
	c.ShowFavourites = fileConfig.ShowFavourites
	c.Filtering = fileConfig.Filtering
	c.RefreshInterval = fileConfig.RefreshInterval

	if fileConfig.HTTPOptions != nil {
		if _, err := TLSVersion(fileConfig.HTTPOptions.MinTLSVersion); err != nil {
			return err
		}
		c.HTTPOptions = fileConfig.HTTPOptions
	}

	if len(fileConfig.Ordering) > 0 {
		c.Ordering = fileConfig.Ordering
	}

	if len(fileConfig.Theme.ReadIcon) > 0 {
		c.Theme.ReadIcon = fileConfig.Theme.ReadIcon
	}

	if fileConfig.Theme.Glamour != "" {
		c.Theme.Glamour = fileConfig.Theme.Glamour
	}

	if fileConfig.Theme.SelectedItemColor != "" {
		c.Theme.SelectedItemColor = fileConfig.Theme.SelectedItemColor
	}

	if fileConfig.Theme.TitleColor != "" {
		c.Theme.TitleColor = fileConfig.Theme.TitleColor
	}

	if fileConfig.Theme.TitleColorFg != "" {
		c.Theme.TitleColorFg = fileConfig.Theme.TitleColorFg
	}

	if fileConfig.Theme.FilterColor != "" {
		c.Theme.FilterColor = fileConfig.Theme.FilterColor
	}

	// only set pager if it's not defined already, config file is lower
	// precidence than flags/env that can be passed to New
	if c.Pager == "" {
		c.Pager = fileConfig.Pager
	}

	if fileConfig.Backends != nil {
		if fileConfig.Backends.Miniflux != nil {
			mffeeds, err := getMinifluxFeeds(fileConfig.Backends.Miniflux)
			if err != nil {
				return err
			}

			c.Feeds = append(c.Feeds, mffeeds...)
		}

		if fileConfig.Backends.FreshRSS != nil {
			freshfeeds, err := getFreshRSSFeeds(fileConfig.Backends.FreshRSS)
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

	// if config file doesn't exist and Create is false, return error
	if !c.Create {
		return fmt.Errorf("config file does not exist for profile '%s'. Use --create to create it", c.Profile)
	}

	// if not, create directory. noop if directory exists
	err = os.MkdirAll(c.ConfigDir, 0755)
	if err != nil {
		return fmt.Errorf("setupConfigDir: %w", err)
	}

	// then create the file
	_, err = os.Create(c.ConfigPath)
	if err != nil {
		return fmt.Errorf("setupConfigDir: %w", err)
	}

	return err
}
