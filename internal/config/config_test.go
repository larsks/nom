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
const configPath = "../test/data/nom/config.yml"

func cleanup() {
	os.RemoveAll(configDir)
}

func TestNewDefault(t *testing.T) {
	c := New()
	ucd, _ := os.UserConfigDir()

	test.Equal(t, fmt.Sprintf("%s/nom/nom.yml", ucd), c.ConfigPath, "Wrong defaults set")
	test.Equal(t, fmt.Sprintf("%s/nom", ucd), c.ConfigDir, "Wrong default ConfigDir set")
	test.Equal(t, "nom", c.Profile, "Wrong default profile")
}

func TestConfigCustomPath(t *testing.T) {
	c := New().WithProfile("bar").WithConfigPath("foo")

	test.Equal(t, "foo/bar.yml", c.ConfigPath, "Config path override not set")
	test.Equal(t, "foo/", c.ConfigDir, "ConfigDir not correctly set")
}

func TestConfigDir(t *testing.T) {
	c := New().WithConfigPath("foo/bizzle")

	test.Equal(t, "foo/bizzle/", c.ConfigDir, "ConfigDir not correctly parsed")
}

func TestNewOverride(t *testing.T) {
	c := New().WithConfigPath("foobar")

	test.Equal(t, "foobar/nom.yml", c.ConfigPath, "Override not respected")
	test.Equal(t, "foobar/", c.ConfigDir, "ConfigDir not correctly set")
}

func TestPreviewFeedsOverrideFeedsFromConfigFile(t *testing.T) {
	configDir, configFile := filepath.Split(configFixturePath)
	// Strip .yml extension from filename to get profile name
	profile := configFile[:len(configFile)-len(filepath.Ext(configFile))]
	c := New().WithProfile(profile).WithConfigPath(configDir)
	c.Load()
	feeds := c.GetFeeds()
	test.Equal(t, 3, len(feeds), "Incorrect feeds number")
	test.Equal(t, "cattle", feeds[0].URL, "First feed in a config must be cattle")
	test.Equal(t, "bird", feeds[1].URL, "Second feed in a config must be bird")
	test.Equal(t, "dog", feeds[2].URL, "Third feed in a config must be dog")

	c = New().WithProfile(profile).WithConfigPath(configDir).WithPreviewFeeds([]string{"pumpkin", "radish"})
	c.Load()
	feeds = c.GetFeeds()
	test.Equal(t, 2, len(feeds), "Incorrect feeds number")
	test.Equal(t, "pumpkin", feeds[0].URL, "First feed in a config must be pumpkin")
	test.Equal(t, "radish", feeds[1].URL, "Second feed in a config must be radish")
}

func TestConfigLoad(t *testing.T) {
	configDir, configFile := filepath.Split(configFixturePath)
	// Strip .yml extension from filename to get profile name
	profile := configFile[:len(configFile)-len(filepath.Ext(configFile))]
	c := New().WithProfile(profile).WithConfigPath(configDir)
	err := c.Load()
	if err != nil {
		t.Fatalf(err.Error())
	}

	if len(c.Feeds) != 3 || c.Feeds[0].URL != "cattle" {
		t.Fatalf("Parsing failed")
	}

	if len(c.Ordering) == 0 || c.Ordering != "desc" {
		t.Fatalf("Parsing failed")
	}
}

func TestConfigLoadFromDirectory(t *testing.T) {
	err := os.MkdirAll(configDir, 0755)
	defer cleanup()

	if err != nil {
		t.Fatalf(err.Error())
	}
	c := New().WithProfile("config").WithConfigPath(configDir)
	if err != nil {
		t.Fatalf(err.Error())
	}

	if c.ConfigPath != configPath {
		t.Fatalf("Failed to find config file in directory")
	}
}

func TestConfigLoadPrecidence(t *testing.T) {
	configDir, configFile := filepath.Split(configFixturePath)
	// Strip .yml extension from filename to get profile name
	profile := configFile[:len(configFile)-len(filepath.Ext(configFile))]
	c := New().WithProfile(profile).WithConfigPath(configDir).WithPager("testpager")

	err := c.Load()
	if err != nil {
		t.Fatalf(err.Error())
	}

	if c.Pager != "testpager" {
		t.Fatalf("testpager overridden")
	}
}

func TestConfigAddFeed(t *testing.T) {
	configDir, configFile := filepath.Split(configFixtureWritePath)
	// Strip .yml extension from filename to get profile name
	profile := configFile[:len(configFile)-len(filepath.Ext(configFile))]
	c := New().WithProfile(profile).WithConfigPath(configDir)

	err := c.Load()
	if err != nil {
		t.Fatalf(err.Error())
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
