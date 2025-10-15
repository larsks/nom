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
	ErrFeedAlreadyExists  = errors.New("config.AddFeed: feed already exists")
	ErrIncludeLoop        = errors.New("config.Load: include loop detected")
	DefaultConfigDirName  = "nom"
	DefaultConfigFileName = "config.yml"
	DefaultDatabaseName   = "nom.db"
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

// Config contains YAML-serializable configuration settings
type Config struct {
	ShowFavourites  bool               `yaml:"showfavourites,omitempty"`
	Pager           string             `yaml:"pager,omitempty"`
	Feeds           []Feed             `yaml:"feeds"`
	Database        string             `yaml:"database"`
	Ordering        constants.Ordering `yaml:"ordering"`
	Filtering       FilterConfig       `yaml:"filtering"`
	Backends        *Backends          `yaml:"backends,omitempty"`
	ShowRead        bool               `yaml:"showread,omitempty"`
	AutoRead        bool               `yaml:"autoread,omitempty"`
	Openers         []Opener           `yaml:"openers,omitempty"`
	Theme           Theme              `yaml:"theme,omitempty"`
	HTTPOptions     *HTTPOptions       `yaml:"http,omitempty"`
	RefreshInterval int                `yaml:"refreshinterval,omitempty"`
	Include         []string           `yaml:"include,omitempty"`
}

// Runtime contains non-serializable runtime settings and the YAML config
type Runtime struct {
	ConfigPath   string
	ConfigDir    string
	PreviewFeeds []Feed
	Version      string
	Config       *Config
}

var DefaultTheme = Theme{
	Glamour:           "dark",
	SelectedItemColor: "170",
	TitleColor:        "62",
	TitleColorFg:      "231",
	FilterColor:       "62",
	ReadIcon:          "\u2713",
}

func (r *Runtime) ToggleShowRead() {
	r.Config.ShowRead = !r.Config.ShowRead
}

func (r *Runtime) ToggleShowFavourites() {
	r.Config.ShowFavourites = !r.Config.ShowFavourites
}

func updateConfigPathIfDir(configPath string) string {
	stat, err := os.Stat(configPath)
	if err == nil && stat.IsDir() {
		configPath = filepath.Join(configPath, DefaultConfigFileName)
	}

	return configPath
}

func New() *Runtime {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		userConfigDir = ""
	}

	configPath := filepath.Join(userConfigDir, DefaultConfigDirName, DefaultConfigFileName)
	configDir, _ := filepath.Split(configPath)

	return &Runtime{
		ConfigPath:   configPath,
		ConfigDir:    configDir,
		PreviewFeeds: []Feed{},
		Version:      "",
		Config: &Config{
			Pager:           "",
			Database:        DefaultDatabaseName,
			Feeds:           []Feed{},
			Theme:           DefaultTheme,
			RefreshInterval: 0,
			Ordering:        constants.DefaultOrdering,
			Filtering: FilterConfig{
				DefaultIncludeFeedName: false,
			},
			HTTPOptions: &HTTPOptions{
				MinTLSVersion: tls.VersionName(tls.VersionTLS12),
			},
		},
	}
}

func (r *Runtime) WithConfigPath(configPath string) *Runtime {
	if configPath != "" {
		r.ConfigPath = updateConfigPathIfDir(configPath)
		r.ConfigDir, _ = filepath.Split(r.ConfigPath)
	}
	return r
}

func (r *Runtime) WithPager(pager string) *Runtime {
	if pager != "" {
		r.Config.Pager = pager
	}
	return r
}

func (r *Runtime) WithPreviewFeeds(previewFeeds []string) *Runtime {
	if len(previewFeeds) > 0 {
		var f []Feed
		for _, feedURL := range previewFeeds {
			f = append(f, Feed{URL: feedURL})
		}
		r.PreviewFeeds = f
	}
	return r
}

func (r *Runtime) WithVersion(version string) *Runtime {
	r.Version = version
	return r
}

func (r *Runtime) WithDatabase(database string) *Runtime {
	if database != "" {
		r.Config.Database = database
	}
	return r
}

func (r *Runtime) IsPreviewMode() bool {
	return len(r.PreviewFeeds) > 0
}

// resolveIncludePath resolves an include path relative to the config directory
// if it's not an absolute path
func resolveIncludePath(configDir, includePath string) string {
	if filepath.IsAbs(includePath) {
		return includePath
	}
	return filepath.Join(configDir, includePath)
}

// loadConfigFile loads a single config file and returns the parsed Config
func loadConfigFile(path string) (*Config, error) {
	rawData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("config.loadConfigFile: %w", err)
	}

	var cfg Config
	err = yaml.Unmarshal(rawData, &cfg)
	if err != nil {
		return nil, fmt.Errorf("config.loadConfigFile: %w", err)
	}

	return &cfg, nil
}

// mergeConfig merges src config into dst config, with src values taking precedence
func mergeConfig(dst, src *Config) {
	// Only override if src has non-zero values
	if src.ShowRead {
		dst.ShowRead = src.ShowRead
	}
	if src.AutoRead {
		dst.AutoRead = src.AutoRead
	}
	if len(src.Feeds) > 0 {
		dst.Feeds = src.Feeds
	}
	if src.Database != "" {
		dst.Database = src.Database
	}
	if len(src.Openers) > 0 {
		dst.Openers = src.Openers
	}
	if src.ShowFavourites {
		dst.ShowFavourites = src.ShowFavourites
	}
	if src.Filtering.DefaultIncludeFeedName {
		dst.Filtering = src.Filtering
	}
	if src.RefreshInterval != 0 {
		dst.RefreshInterval = src.RefreshInterval
	}
	if src.HTTPOptions != nil {
		dst.HTTPOptions = src.HTTPOptions
	}
	if len(src.Ordering) > 0 {
		dst.Ordering = src.Ordering
	}
	if len(src.Theme.ReadIcon) > 0 {
		dst.Theme.ReadIcon = src.Theme.ReadIcon
	}
	if src.Theme.Glamour != "" {
		dst.Theme.Glamour = src.Theme.Glamour
	}
	if src.Theme.SelectedItemColor != "" {
		dst.Theme.SelectedItemColor = src.Theme.SelectedItemColor
	}
	if src.Theme.TitleColor != "" {
		dst.Theme.TitleColor = src.Theme.TitleColor
	}
	if src.Theme.TitleColorFg != "" {
		dst.Theme.TitleColorFg = src.Theme.TitleColorFg
	}
	if src.Theme.FilterColor != "" {
		dst.Theme.FilterColor = src.Theme.FilterColor
	}
	if src.Pager != "" {
		dst.Pager = src.Pager
	}
	if src.Backends != nil {
		dst.Backends = src.Backends
	}
	if len(src.Include) > 0 {
		dst.Include = src.Include
	}
}

// loadConfigWithIncludes recursively loads config files with include support
// visited tracks files already loaded to detect include loops
func (r *Runtime) loadConfigWithIncludes(configPath string, visited map[string]bool) (*Config, error) {
	// Normalize path for loop detection
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("config.loadConfigWithIncludes: %w", err)
	}

	// Check for include loops
	if visited[absPath] {
		return nil, ErrIncludeLoop
	}
	visited[absPath] = true

	// Load the config file
	cfg, err := loadConfigFile(configPath)
	if err != nil {
		return nil, err
	}

	// Process includes in order
	if len(cfg.Include) > 0 {
		configDir := filepath.Dir(configPath)
		baseConfig := &Config{
			Database:  DefaultDatabaseName,
			Theme:     DefaultTheme,
			Ordering:  constants.DefaultOrdering,
			Filtering: FilterConfig{DefaultIncludeFeedName: false},
			HTTPOptions: &HTTPOptions{
				MinTLSVersion: tls.VersionName(tls.VersionTLS12),
			},
		}

		for _, includePath := range cfg.Include {
			resolvedPath := resolveIncludePath(configDir, includePath)

			includedCfg, err := r.loadConfigWithIncludes(resolvedPath, visited)
			if err != nil {
				return nil, fmt.Errorf("config.loadConfigWithIncludes: error loading %s: %w", includePath, err)
			}

			mergeConfig(baseConfig, includedCfg)
		}

		// Merge the current config on top of all includes
		mergeConfig(baseConfig, cfg)
		cfg = baseConfig
	}

	return cfg, nil
}

func (r *Runtime) Load() error {
	err := r.setupConfigDir()
	if err != nil {
		return fmt.Errorf("config Load: %w", err)
	}

	// Load config with include support
	visited := make(map[string]bool)
	fileConfig, err := r.loadConfigWithIncludes(r.ConfigPath, visited)
	if err != nil {
		return fmt.Errorf("config.Load: %w", err)
	}

	// Validate HTTPOptions if present
	if fileConfig.HTTPOptions != nil {
		if _, err := TLSVersion(fileConfig.HTTPOptions.MinTLSVersion); err != nil {
			return err
		}
	}

	// Merge loaded config with runtime config, respecting priority
	// (command-line flags and builder methods take precedence over file)
	r.Config.ShowRead = fileConfig.ShowRead
	r.Config.AutoRead = fileConfig.AutoRead
	r.Config.Feeds = fileConfig.Feeds
	if fileConfig.Database != "" {
		r.Config.Database = fileConfig.Database
	}
	r.Config.Openers = fileConfig.Openers
	r.Config.ShowFavourites = fileConfig.ShowFavourites
	r.Config.Filtering = fileConfig.Filtering
	r.Config.RefreshInterval = fileConfig.RefreshInterval

	if fileConfig.HTTPOptions != nil {
		r.Config.HTTPOptions = fileConfig.HTTPOptions
	}

	if len(fileConfig.Ordering) > 0 {
		r.Config.Ordering = fileConfig.Ordering
	}

	if len(fileConfig.Theme.ReadIcon) > 0 {
		r.Config.Theme.ReadIcon = fileConfig.Theme.ReadIcon
	}

	if fileConfig.Theme.Glamour != "" {
		r.Config.Theme.Glamour = fileConfig.Theme.Glamour
	}

	if fileConfig.Theme.SelectedItemColor != "" {
		r.Config.Theme.SelectedItemColor = fileConfig.Theme.SelectedItemColor
	}

	if fileConfig.Theme.TitleColor != "" {
		r.Config.Theme.TitleColor = fileConfig.Theme.TitleColor
	}

	if fileConfig.Theme.TitleColorFg != "" {
		r.Config.Theme.TitleColorFg = fileConfig.Theme.TitleColorFg
	}

	if fileConfig.Theme.FilterColor != "" {
		r.Config.Theme.FilterColor = fileConfig.Theme.FilterColor
	}

	// only set pager if it's not defined already, config file is lower
	// precedence than flags/env that can be passed to New
	if r.Config.Pager == "" {
		r.Config.Pager = fileConfig.Pager
	}

	if len(fileConfig.Include) > 0 {
		r.Config.Include = fileConfig.Include
	}

	// Process backends and fetch feeds from external sources
	if fileConfig.Backends != nil {
		if fileConfig.Backends.Miniflux != nil {
			mffeeds, err := getMinifluxFeeds(fileConfig.Backends.Miniflux)
			if err != nil {
				return err
			}

			r.Config.Feeds = append(r.Config.Feeds, mffeeds...)
		}

		if fileConfig.Backends.FreshRSS != nil {
			freshfeeds, err := getFreshRSSFeeds(fileConfig.Backends.FreshRSS)
			if err != nil {
				return err
			}

			r.Config.Feeds = append(r.Config.Feeds, freshfeeds...)
		}
	}

	return nil
}

// Write writes to a config file
func (r *Runtime) Write() error {
	str, err := yaml.Marshal(r.Config)
	if err != nil {
		return fmt.Errorf("config.Write: %w", err)
	}

	err = os.WriteFile(r.ConfigPath, []byte(str), 0655)
	if err != nil {
		return fmt.Errorf("config.Write: %w", err)
	}

	return nil
}

func (r *Runtime) AddFeed(feed Feed) error {
	err := r.Load()
	if err != nil {
		return fmt.Errorf("config.AddFeed: %w", err)
	}

	for _, f := range r.Config.Feeds {
		if f.URL == feed.URL {
			return ErrFeedAlreadyExists
		}
	}

	r.Config.Feeds = append(r.Config.Feeds, feed)

	err = r.Write()
	if err != nil {
		return fmt.Errorf("config.AddFeed: %w", err)
	}

	return nil
}

func (r *Runtime) GetFeeds() []Feed {
	if r.IsPreviewMode() {
		return r.PreviewFeeds
	}

	return r.Config.Feeds
}

func (r *Runtime) setupConfigDir() error {
	_, err := os.Stat(r.ConfigPath)

	// if configFile exists, do nothing
	if !errors.Is(err, os.ErrNotExist) {
		return nil
	}

	// if not, create directory. noop if directory exists
	err = os.MkdirAll(r.ConfigDir, 0755)
	if err != nil {
		return fmt.Errorf("setupConfigDir: %w", err)
	}

	// then create the file
	_, err = os.Create(r.ConfigPath)
	if err != nil {
		return fmt.Errorf("setupConfigDir: %w", err)
	}

	return err
}

func (r *Runtime) ImportFeeds() ([]Feed, error) {
	err := r.Load()
	if err != nil {
		return nil, fmt.Errorf("config.ImportFeeds: %w", err)
	}

	return nil, nil
}
