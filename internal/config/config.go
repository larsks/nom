package config

import (
	"crypto/tls"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"dario.cat/mergo"
	"gopkg.in/yaml.v3"

	"github.com/guyfedwards/nom/v2/internal/constants"
)

//go:embed default_config.yml
var defaultConfig string

var (
	ErrFeedAlreadyExists  = errors.New("config.AddFeed: feed already exists")
	ErrIncludeLoop        = errors.New("config.Load: include loop detected")
	DefaultConfigDirName  = "nom"
	DefaultConfigFileName = "default.yml"
	LegacyConfigFileName  = "config.yml"
	LegacyDatabaseName    = "nom.db"
	DefaultListFormat     = `{{ printf "%3d" .Index }}. {{ if .Item.FeedName }}{{ .Item.FeedName }}: {{ end }}{{ .Item.Title }}`
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
	AutoReadBrowser bool               `yaml:"autoreadbrowser,omitempty"`
	Openers         []Opener           `yaml:"openers,omitempty"`
	Theme           Theme              `yaml:"theme,omitempty"`
	HTTPOptions     *HTTPOptions       `yaml:"http,omitempty"`
	RefreshInterval int                `yaml:"refreshinterval,omitempty"`
	ListFormat      string             `yaml:"listformat,omitempty"`
	Include         []string           `yaml:"include,omitempty"`
	UserAgent       string             `yaml:"useragent,omitempty"`
}

// Runtime contains non-serializable runtime settings and the YAML config
type Runtime struct {
	ConfigPath   string
	ConfigDir    string
	PreviewFeeds []Feed
	Version      string
	Create       bool
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

// defaultDatabaseName derives the default database name from the config file path
// For example: "config.yml" -> "config.db", "/path/to/myconfig.yml" -> "myconfig.db"
func defaultDatabaseName(configPath string) string {
	basename := filepath.Base(configPath)
	// Remove the extension
	ext := filepath.Ext(basename)
	if ext != "" {
		basename = basename[:len(basename)-len(ext)]
	}
	return basename + ".db"
}

func getNomConfigDir() string {
	nomConfigDir := os.Getenv("NOM_CONFIG_DIR")
	if nomConfigDir == "" {
		userConfigDir, err := os.UserConfigDir()
		if err != nil {
			userConfigDir = ""
		}

		nomConfigDir = filepath.Join(userConfigDir, DefaultConfigDirName)
	}

	return nomConfigDir
}

func New() *Runtime {
	configDir := getNomConfigDir()
	configPath := filepath.Join(configDir, DefaultConfigFileName)

	return &Runtime{
		ConfigPath:   configPath,
		ConfigDir:    configDir + string(filepath.Separator),
		PreviewFeeds: []Feed{},
		Version:      "",
		Config: &Config{
			Pager:           "",
			Database:        "", // Will be computed in Load() if not set via WithDatabase()
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
			ListFormat: DefaultListFormat,
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

func (r *Runtime) WithCreate(create bool) *Runtime {
	r.Create = create
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
		baseConfig := &Config{}

		for _, includePath := range cfg.Include {
			resolvedPath := resolveIncludePath(configDir, includePath)

			includedCfg, err := r.loadConfigWithIncludes(resolvedPath, visited)
			if err != nil {
				return nil, fmt.Errorf("config.loadConfigWithIncludes: error loading %s: %w", includePath, err)
			}

			if err := mergo.Merge(baseConfig, includedCfg, mergo.WithOverride); err != nil {
				return nil, fmt.Errorf("config.loadConfigWithIncludes: error merging %s: %w", includePath, err)
			}
		}

		// Merge the current config on top of all includes
		if err := mergo.Merge(baseConfig, cfg, mergo.WithOverride); err != nil {
			return nil, fmt.Errorf("config.loadConfigWithIncludes: error merging base config: %w", err)
		}
		cfg = baseConfig
	}

	return cfg, nil
}

func (r *Runtime) Load() (*Runtime, error) {
	err := r.setupConfigDir()
	if err != nil {
		// Check for legacy config file fallback if using default config path
		// This ensures backwards compatibility when upgrading from older versions
		if filepath.Base(r.ConfigPath) == DefaultConfigFileName {
			r.ConfigPath = filepath.Join(r.ConfigDir, LegacyConfigFileName)
			if fallbackErr := r.setupConfigDir(); fallbackErr == nil {
				err = nil // Fallback succeeded, clear the error
			}
		}
		if err != nil {
			return nil, fmt.Errorf("config Load: %w", err)
		}
	}

	// If we're in preview mode and config file doesn't exist, skip loading
	// and use the defaults from New()
	_, statErr := os.Stat(r.ConfigPath)
	if r.IsPreviewMode() && errors.Is(statErr, os.ErrNotExist) {
		// Just use default config values - no need to load file
		// Database name will be computed below
		if r.Config.Database == "" {
			r.Config.Database = defaultDatabaseName(r.ConfigPath)
		}
		return r, nil
	}

	// Load config with include support
	visited := make(map[string]bool)
	fileConfig, err := r.loadConfigWithIncludes(r.ConfigPath, visited)
	if err != nil {
		return nil, fmt.Errorf("config.Load: %w", err)
	}

	// Validate HTTPOptions if present
	if fileConfig.HTTPOptions != nil {
		if _, err := TLSVersion(fileConfig.HTTPOptions.MinTLSVersion); err != nil {
			return nil, err
		}
	}

	// Store database name in case it was set explicitly
	existingDatabase := r.Config.Database

	// Merge loaded config with runtime config
	// fileConfig values override r.Config defaults (except values set via With*(), handled below)
	if err := mergo.Merge(r.Config, fileConfig, mergo.WithOverride); err != nil {
		return nil, fmt.Errorf("config.Load: error merging config: %w", err)
	}

	// Compute database name from config path if not explicitly set
	if existingDatabase == "" && r.Config.Database == "" {
		// Use legacy database name for legacy config file
		if filepath.Base(r.ConfigPath) == LegacyConfigFileName {
			r.Config.Database = LegacyDatabaseName
		} else {
			r.Config.Database = defaultDatabaseName(r.ConfigPath)
		}
	} else if existingDatabase != "" {
		// Restore database if it was set via WithDatabase() (higher precedence than file)
		r.Config.Database = existingDatabase
	}

	// Process backends and fetch feeds from external sources
	if fileConfig.Backends != nil {
		if fileConfig.Backends.Miniflux != nil {
			mffeeds, err := getMinifluxFeeds(fileConfig.Backends.Miniflux)
			if err != nil {
				return nil, err
			}

			r.Config.Feeds = append(r.Config.Feeds, mffeeds...)
		}

		if fileConfig.Backends.FreshRSS != nil {
			freshfeeds, err := getFreshRSSFeeds(fileConfig.Backends.FreshRSS)
			if err != nil {
				return nil, err
			}

			r.Config.Feeds = append(r.Config.Feeds, freshfeeds...)
		}
	}

	return r, nil
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
	_, err := r.Load()
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

	// we don't need a configuration if we're running in preview mode
	// (we'll use default config values and feeds from the command line)
	if r.IsPreviewMode() {
		return nil
	}

	// if config file doesn't exist and Create flag is false, return error
	if !r.Create {
		return fmt.Errorf("setupConfigDir: config file does not exist: %s (use --create to create it)", r.ConfigPath)
	}

	// if not, create directory. noop if directory exists
	err = os.MkdirAll(r.ConfigDir, 0755)
	if err != nil {
		return fmt.Errorf("setupConfigDir: %w", err)
	}

	// then create the file
	err = os.WriteFile(r.ConfigPath, []byte(defaultConfig), 0755)
	if err != nil {
		return fmt.Errorf("setupConfigDir: %w", err)
	}

	return err
}

func (r *Runtime) ImportFeeds() ([]Feed, error) {
	_, err := r.Load()
	if err != nil {
		return nil, fmt.Errorf("config.ImportFeeds: %w", err)
	}

	return nil, nil
}
