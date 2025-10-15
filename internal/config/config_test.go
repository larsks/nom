package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/guyfedwards/nom/v2/internal/test"
)

const configFixturePath = "../test/data/config_fixture.yml"
const configFixtureWritePath = "../test/data/config_fixture_write.yml"
const configDir = "../test/data/nom"
const configPath = "../test/data/nom/default.yml"

func cleanup() {
	os.RemoveAll(configDir)
}

func TestNewDefault(t *testing.T) {
	c, err := New().WithCreate(true).Load()
	if err != nil {
		t.Fatalf("Failed to load config: %s", err)
	}
	ucd, _ := os.UserConfigDir()

	test.Equal(t, fmt.Sprintf("%s/nom/default.yml", ucd), c.ConfigPath, "Wrong defaults set")
	test.Equal(t, fmt.Sprintf("%s/nom/", ucd), c.ConfigDir, "Wrong default ConfigDir set")
	test.Equal(t, "default.db", c.Config.Database, "Wrong default database name")
}

func TestConfigCustomPath(t *testing.T) {
	c := New().WithConfigPath("foo/bar.yml")

	test.Equal(t, "foo/bar.yml", c.ConfigPath, "Config path override not set")
}

func TestConfigDir(t *testing.T) {
	c := New().WithConfigPath("foo/bizzle/bar.yml")

	test.Equal(t, "foo/bizzle/", c.ConfigDir, "ConfigDir not correctly parsed")
}

func TestNewOverride(t *testing.T) {
	c := New().WithConfigPath("foobar")

	test.Equal(t, "foobar", c.ConfigPath, "Override not respected")
}

func TestPreviewFeedsOverrideFeedsFromConfigFile(t *testing.T) {
	c, err := New().WithConfigPath(configFixturePath).Load()
	if err != nil {
		t.Fatalf("Failed to load config: %s", err)
	}
	feeds := c.GetFeeds()
	test.Equal(t, 3, len(feeds), "Incorrect feeds number")
	test.Equal(t, "cattle", feeds[0].URL, "First feed in a config must be cattle")
	test.Equal(t, "bird", feeds[1].URL, "Second feed in a config must be bird")
	test.Equal(t, "dog", feeds[2].URL, "Third feed in a config must be dog")

	c, err = New().WithConfigPath(configFixturePath).WithPreviewFeeds([]string{"pumpkin", "radish"}).Load()
	if err != nil {
		t.Fatalf("Failed to load config: %s", err)
	}
	feeds = c.GetFeeds()
	test.Equal(t, 2, len(feeds), "Incorrect feeds number")
	test.Equal(t, "pumpkin", feeds[0].URL, "First feed in a config must be pumpkin")
	test.Equal(t, "radish", feeds[1].URL, "Second feed in a config must be radish")
}

func TestConfigLoad(t *testing.T) {
	c, err := New().WithConfigPath(configFixturePath).Load()
	if err != nil {
		t.Fatalf("%s", err)
	}

	if len(c.Config.Feeds) != 3 || c.Config.Feeds[0].URL != "cattle" {
		t.Fatalf("Parsing failed")
	}

	if len(c.Config.Ordering) == 0 || c.Config.Ordering != "desc" {
		t.Fatalf("Parsing failed")
	}
}

func TestConfigLoadFromDirectory(t *testing.T) {
	err := os.MkdirAll(configDir, 0755)
	defer cleanup()

	if err != nil {
		t.Fatalf("%s", err)
	}
	c := New().WithConfigPath(configDir)
	if err != nil {
		t.Fatalf("%s", err)
	}

	if c.ConfigPath != configPath {
		t.Fatalf("Failed to find config file in directory")
	}
}

func TestConfigLoadPrecidence(t *testing.T) {
	c, err := New().WithConfigPath(configFixturePath).WithPager("testpager").Load()
	if err != nil {
		t.Fatalf("%s", err)
	}

	if c.Config.Pager != "testpager" {
		t.Fatalf("testpager overridden")
	}
}

func TestConfigAddFeed(t *testing.T) {
	c, err := New().WithConfigPath(configFixtureWritePath).Load()
	if err != nil {
		t.Fatalf("%s", err)
	}

	c.AddFeed(Feed{URL: "foo"})

	var actual Config
	rawData, _ := os.ReadFile(c.ConfigPath)
	_ = yaml.Unmarshal(rawData, &actual)

	hasAdded := false
	for _, v := range actual.Feeds {
		if v.URL == "newfeed" {
			hasAdded = true
			break
		}
	}

	if !hasAdded {
		t.Fatalf("did not write feed correctly")
	}
}
func TestConfigSetupDir(t *testing.T) {
	err := os.MkdirAll(configDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create %s", configDir)
	}

	_, err = New().WithConfigPath(configPath).WithCreate(true).Load()
	if err != nil {
		t.Fatalf("Failed to load config: %s", err)
	}

	_, err = os.Stat(configPath)
	if err != nil {
		t.Fatalf("Did not create %s as expected", configPath)
	}

	cleanup()
}

func TestIncludeBasic(t *testing.T) {
	c, err := New().WithConfigPath("../test/data/include_main.yml").Load()
	if err != nil {
		t.Fatalf("Failed to load config with includes: %s", err)
	}

	// Main config should override included configs
	if len(c.Config.Feeds) != 1 || c.Config.Feeds[0].URL != "main-feed" {
		t.Fatalf("Expected main-feed, got %v", c.Config.Feeds)
	}

	// Ordering from include_override.yml should be present
	if c.Config.Ordering != "desc" {
		t.Fatalf("Expected ordering 'desc', got %s", c.Config.Ordering)
	}

	// TitleColor from main config should override include_base.yml
	if c.Config.Theme.TitleColor != "200" {
		t.Fatalf("Expected titleColor '200', got %s", c.Config.Theme.TitleColor)
	}

	// Pager from include_override.yml should be present
	if c.Config.Pager != "less" {
		t.Fatalf("Expected pager 'less', got %s", c.Config.Pager)
	}
}

func TestIncludeLoop(t *testing.T) {
	_, err := New().WithConfigPath("../test/data/include_loop_a.yml").Load()
	if err == nil {
		t.Fatalf("Expected error for include loop, got nil")
	}

	// Check if the error chain contains ErrIncludeLoop
	var found bool
	for e := err; e != nil; {
		if e == ErrIncludeLoop {
			found = true
			break
		}
		// Unwrap if possible
		if unwrapper, ok := e.(interface{ Unwrap() error }); ok {
			e = unwrapper.Unwrap()
		} else {
			break
		}
	}

	if !found {
		t.Fatalf("Expected ErrIncludeLoop in error chain, got %v", err)
	}
}

func TestIncludeNested(t *testing.T) {
	c, err := New().WithConfigPath("../test/data/include_nested_level1.yml").Load()
	if err != nil {
		t.Fatalf("Failed to load nested config: %s", err)
	}

	// Level 1 feeds should override level 2
	if len(c.Config.Feeds) != 1 || c.Config.Feeds[0].URL != "level1-feed" {
		t.Fatalf("Expected level1-feed, got %v", c.Config.Feeds)
	}

	// Level 1 ordering should override level 2
	if c.Config.Ordering != "desc" {
		t.Fatalf("Expected ordering 'desc', got %s", c.Config.Ordering)
	}

	// Pager from level 2 should still be present
	if c.Config.Pager != "cat" {
		t.Fatalf("Expected pager 'cat', got %s", c.Config.Pager)
	}
}

func TestIncludeMissingFile(t *testing.T) {
	// Create a temporary config file with a missing include
	tmpfile, err := os.CreateTemp("", "config_*.yml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %s", err)
	}
	defer os.Remove(tmpfile.Name())

	content := []byte("include:\n  - nonexistent.yml\nfeeds:\n  - url: test\n")
	if _, err := tmpfile.Write(content); err != nil {
		t.Fatalf("Failed to write temp file: %s", err)
	}
	tmpfile.Close()

	_, err = New().WithConfigPath(tmpfile.Name()).Load()
	if err == nil {
		t.Fatalf("Expected error for missing include file, got nil")
	}
}

func TestResolveIncludePath(t *testing.T) {
	// Test relative path
	result := resolveIncludePath("/home/user/config", "subdir/file.yml")
	expected := "/home/user/config/subdir/file.yml"
	if result != expected {
		t.Fatalf("Expected %s, got %s", expected, result)
	}

	// Test absolute path
	result = resolveIncludePath("/home/user/config", "/etc/nom/file.yml")
	expected = "/etc/nom/file.yml"
	if result != expected {
		t.Fatalf("Expected %s, got %s", expected, result)
	}
}

func TestConfigSetupDirWithoutCreateFlag(t *testing.T) {
	// Clean up first to ensure the file doesn't exist
	os.RemoveAll(configDir)
	defer cleanup()

	// Try to load a config file that doesn't exist without the Create flag
	_, err := New().WithConfigPath(configPath).WithCreate(false).Load()

	// Should get an error because the file doesn't exist and Create is false
	if err == nil {
		t.Fatalf("Expected error when config file doesn't exist and Create is false, got nil")
	}

	// Verify the error message mentions using --create
	expectedMsg := "use --create to create it"
	if !containsString(err.Error(), expectedMsg) {
		t.Fatalf("Expected error message to contain '%s', got: %s", expectedMsg, err.Error())
	}

	// Verify the file was not created
	_, err = os.Stat(configPath)
	if err == nil {
		t.Fatalf("Config file should not have been created")
	}
}

func TestConfigSetupDirWithCreateFlag(t *testing.T) {
	// Clean up first to ensure the file doesn't exist
	os.RemoveAll(configDir)
	defer cleanup()

	// Create the config file with the Create flag
	_, err := New().WithConfigPath(configPath).WithCreate(true).Load()
	if err != nil {
		t.Fatalf("Expected no error when Create is true, got: %s", err)
	}

	// Verify the file was created
	_, err = os.Stat(configPath)
	if err != nil {
		t.Fatalf("Config file should have been created, got error: %s", err)
	}
}

func TestConfigExistingFileWithoutCreateFlag(t *testing.T) {
	// Use an existing config file
	c, err := New().WithConfigPath(configFixturePath).WithCreate(false).Load()

	// Should work fine because the file exists
	if err != nil {
		t.Fatalf("Expected no error when config file exists, got: %s", err)
	}

	// Verify feeds were loaded correctly
	if len(c.Config.Feeds) != 3 {
		t.Fatalf("Expected 3 feeds, got %d", len(c.Config.Feeds))
	}
}

func TestDefaultDatabaseName(t *testing.T) {
	tests := []struct {
		configPath  string
		expectedDB  string
		description string
	}{
		{"config.yml", "config.db", "Simple config file"},
		{"/path/to/myconfig.yml", "myconfig.db", "Full path config file"},
		{"custom.yml", "custom.db", "Custom config file"},
		{"default.yml", "default.db", "Default config file"},
		{"/home/user/.config/nom/feeds.yml", "feeds.db", "Absolute path"},
	}

	for _, tt := range tests {
		result := defaultDatabaseName(tt.configPath)
		if result != tt.expectedDB {
			t.Errorf("%s: expected %s, got %s", tt.description, tt.expectedDB, result)
		}
	}
}

func TestConfigPathDeterminesDatabaseName(t *testing.T) {
	// Test that when a custom config path is set, the database name is derived from it after Load()
	c, err := New().WithConfigPath(configFixturePath).Load()
	if err != nil {
		t.Fatalf("Failed to load config: %s", err)
	}
	test.Equal(t, configFixturePath, c.ConfigPath, "Config path not set correctly")
	test.Equal(t, "config_fixture.db", c.Config.Database, "Database name should be derived from config path")

	// Test that WithDatabase() takes precedence over computed default
	c2, err := New().WithConfigPath(configFixturePath).WithDatabase("custom.db").Load()
	if err != nil {
		t.Fatalf("Failed to load config: %s", err)
	}
	test.Equal(t, "custom.db", c2.Config.Database, "WithDatabase should override computed default")
}

func TestLegacyConfigFileBackwardsCompatibility(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "nom-legacy-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	nomDir := filepath.Join(tmpDir, "nom")
	err = os.MkdirAll(nomDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create nom dir: %s", err)
	}

	// Create only the legacy config.yml file
	legacyConfigPath := filepath.Join(nomDir, "config.yml")
	legacyContent := []byte("feeds:\n  - url: legacy-feed\n")
	err = os.WriteFile(legacyConfigPath, legacyContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write legacy config: %s", err)
	}

	// Create a Runtime with default config path pointing to default.yml (which doesn't exist)
	defaultConfigPath := filepath.Join(nomDir, "default.yml")
	c, err := New().WithConfigPath(defaultConfigPath).Load()
	if err != nil {
		t.Fatalf("Failed to load config with legacy fallback: %s", err)
	}

	// Verify it fell back to config.yml
	test.Equal(t, legacyConfigPath, c.ConfigPath, "Should have fallen back to legacy config.yml")

	// Verify it loaded the legacy config content
	test.Equal(t, 1, len(c.Config.Feeds), "Should have loaded one feed")
	test.Equal(t, "legacy-feed", c.Config.Feeds[0].URL, "Should have loaded legacy feed")
}

func TestLegacyDatabaseName(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "nom-legacy-db-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	nomDir := filepath.Join(tmpDir, "nom")
	err = os.MkdirAll(nomDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create nom dir: %s", err)
	}

	// Create only the legacy config.yml file
	legacyConfigPath := filepath.Join(nomDir, "config.yml")
	legacyContent := []byte("feeds:\n  - url: test-feed\n")
	err = os.WriteFile(legacyConfigPath, legacyContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write legacy config: %s", err)
	}

	// Create a Runtime with default config path
	defaultConfigPath := filepath.Join(nomDir, "default.yml")
	c, err := New().WithConfigPath(defaultConfigPath).Load()
	if err != nil {
		t.Fatalf("Failed to load config: %s", err)
	}

	// Verify it uses nom.db as the database name (not config.db)
	test.Equal(t, "nom.db", c.Config.Database, "Should use legacy database name nom.db")
}

func TestNewDefaultConfigTakesPrecedence(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "nom-precedence-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	nomDir := filepath.Join(tmpDir, "nom")
	err = os.MkdirAll(nomDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create nom dir: %s", err)
	}

	// Create both default.yml and config.yml
	defaultConfigPath := filepath.Join(nomDir, "default.yml")
	defaultContent := []byte("feeds:\n  - url: new-default-feed\n")
	err = os.WriteFile(defaultConfigPath, defaultContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write default config: %s", err)
	}

	legacyConfigPath := filepath.Join(nomDir, "config.yml")
	legacyContent := []byte("feeds:\n  - url: legacy-feed\n")
	err = os.WriteFile(legacyConfigPath, legacyContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write legacy config: %s", err)
	}

	// Load with default path
	c, err := New().WithConfigPath(defaultConfigPath).Load()
	if err != nil {
		t.Fatalf("Failed to load config: %s", err)
	}

	// Verify it uses default.yml, not config.yml
	test.Equal(t, defaultConfigPath, c.ConfigPath, "Should use default.yml when it exists")
	test.Equal(t, "new-default-feed", c.Config.Feeds[0].URL, "Should load from default.yml")
	test.Equal(t, "default.db", c.Config.Database, "Should use default.db, not nom.db")
}

func TestExplicitConfigPathIgnoresLegacy(t *testing.T) {
	// Create a temporary directory structure
	tmpDir, err := os.MkdirTemp("", "nom-explicit-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %s", err)
	}
	defer os.RemoveAll(tmpDir)

	nomDir := filepath.Join(tmpDir, "nom")
	err = os.MkdirAll(nomDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create nom dir: %s", err)
	}

	// Create a custom config file
	customConfigPath := filepath.Join(nomDir, "custom.yml")
	customContent := []byte("feeds:\n  - url: custom-feed\n")
	err = os.WriteFile(customConfigPath, customContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write custom config: %s", err)
	}

	// Also create a legacy config.yml
	legacyConfigPath := filepath.Join(nomDir, "config.yml")
	legacyContent := []byte("feeds:\n  - url: legacy-feed\n")
	err = os.WriteFile(legacyConfigPath, legacyContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write legacy config: %s", err)
	}

	// Load with explicit custom path
	c, err := New().WithConfigPath(customConfigPath).Load()
	if err != nil {
		t.Fatalf("Failed to load config: %s", err)
	}

	// Verify it uses the explicit custom path, not the legacy fallback
	test.Equal(t, customConfigPath, c.ConfigPath, "Should use explicitly set config path")
	test.Equal(t, "custom-feed", c.Config.Feeds[0].URL, "Should load from custom config")
	test.Equal(t, "custom.db", c.Config.Database, "Should derive database name from custom config")
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
