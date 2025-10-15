package config

import (
	"fmt"
	"os"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/guyfedwards/nom/v2/internal/test"
)

const configFixturePath = "../test/data/config_fixture.yml"
const configFixtureWritePath = "../test/data/config_fixture_write.yml"
const configDir = "../test/data/nom"
const configPath = "../test/data/nom/config.yml"

func cleanup() {
	os.RemoveAll(configDir)
}

func TestNewDefault(t *testing.T) {
	c := New()
	ucd, _ := os.UserConfigDir()

	test.Equal(t, fmt.Sprintf("%s/nom/config.yml", ucd), c.ConfigPath, "Wrong defaults set")
	test.Equal(t, fmt.Sprintf("%s/nom/", ucd), c.ConfigDir, "Wrong default ConfigDir set")
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
	c := New().WithConfigPath(configFixturePath)
	c.Load()
	feeds := c.GetFeeds()
	test.Equal(t, 3, len(feeds), "Incorrect feeds number")
	test.Equal(t, "cattle", feeds[0].URL, "First feed in a config must be cattle")
	test.Equal(t, "bird", feeds[1].URL, "Second feed in a config must be bird")
	test.Equal(t, "dog", feeds[2].URL, "Third feed in a config must be dog")

	c = New().WithConfigPath(configFixturePath).WithPreviewFeeds([]string{"pumpkin", "radish"})
	c.Load()
	feeds = c.GetFeeds()
	test.Equal(t, 2, len(feeds), "Incorrect feeds number")
	test.Equal(t, "pumpkin", feeds[0].URL, "First feed in a config must be pumpkin")
	test.Equal(t, "radish", feeds[1].URL, "Second feed in a config must be radish")
}

func TestConfigLoad(t *testing.T) {
	c := New().WithConfigPath(configFixturePath)
	err := c.Load()
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
	c := New().WithConfigPath(configFixturePath).WithPager("testpager")

	err := c.Load()
	if err != nil {
		t.Fatalf("%s", err)
	}

	if c.Config.Pager != "testpager" {
		t.Fatalf("testpager overridden")
	}
}

func TestConfigAddFeed(t *testing.T) {
	c := New().WithConfigPath(configFixtureWritePath)

	err := c.Load()
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

	c := New().WithConfigPath(configPath)
	c.Load()

	_, err = os.Stat(configPath)
	if err != nil {
		t.Fatalf("Did not create %s as expected", configPath)
	}

	cleanup()
}

func TestIncludeBasic(t *testing.T) {
	c := New().WithConfigPath("../test/data/include_main.yml")
	err := c.Load()
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
	c := New().WithConfigPath("../test/data/include_loop_a.yml")
	err := c.Load()
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
	c := New().WithConfigPath("../test/data/include_nested_level1.yml")
	err := c.Load()
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

	c := New().WithConfigPath(tmpfile.Name())
	err = c.Load()
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
