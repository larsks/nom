package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"

	"github.com/guyfedwards/nom/v2/internal/commands"
	"github.com/guyfedwards/nom/v2/internal/config"
	store "github.com/guyfedwards/nom/v2/internal/store/badgerstore"
)

type Options struct {
	Verbose      bool     `short:"v" long:"verbose" description:"Show verbose logging"`
	Pager        string   `short:"p" long:"pager" description:"Pager to use for longer output. Set to false for no pager"`
	ConfigDir    string   `short:"c" long:"config-dir" description:"Directory containing config files" env:"NOM_CONFIG_DIR"`
	DataDir      string   `short:"d" long:"data-dir" description:"Directory for storing data" env:"NOM_DATA_DIR"`
	Profile      string   `short:"P" long:"profile" description:"Profile name (determines config file and database name)" env:"NOM_PROFILE"`
	Create       bool     `long:"create" description:"Create config file and directories if they don't exist"`
	PreviewFeeds []string `short:"f" long:"feed" description:"Feed(s) URL(s) for preview"`
}

var (
	options Options
	version = "dev"
)

// Setup subcommands

type Add struct {
	Positional struct {
		Url  string `positional-arg-name:"URL" required:"yes"`
		Name string `positional-arg-name:"NAME"`
	} `positional-args:"yes"`
}

func (r *Add) Execute(args []string) error {
	cmds, err := getCmds()
	if err != nil {
		return err
	}
	return cmds.Add(r.Positional.Url, r.Positional.Name)
}

type Config struct{}

func (r *Config) Execute(args []string) error {
	cmds, err := getCmds()
	if err != nil {
		return err
	}
	return cmds.ShowConfig()
}

type List struct{}

func (r *List) Execute(args []string) error {
	cmds, err := getCmds()
	if err != nil {
		return err
	}

	return cmds.List()
}

type Version struct{}

func (r *Version) Execute(args []string) error {
	_, err := getCmds()
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", version)
	return nil
}

type Refresh struct{}

func (r *Refresh) Execute(args []string) error {
	cmds, err := getCmds()
	if err != nil {
		return err
	}
	return cmds.Refresh()
}

type Unread struct{}

func (r *Unread) Execute(args []string) error {
	cmds, err := getCmds()
	if err != nil {
		return err
	}
	count := cmds.CountUnread()
	fmt.Printf("%d\n", count)
	return nil
}

func getCmds() (*commands.Commands, error) {
	profile := options.Profile
	if profile == "" {
		profile = "nom"
	}

	cfg := config.New().
		WithProfile(profile).
		WithConfigPath(options.ConfigDir).
		WithDataDir(options.DataDir).
		WithPager(options.Pager).
		WithPreviewFeeds(options.PreviewFeeds).
		WithCreate(options.Create).
		WithVersion(version)

	if err := cfg.Load(); err != nil {
		return nil, err
	}

	s, err := store.NewBadgerStore(cfg.CacheDir, cfg.Database)
	if err != nil {
		return nil, fmt.Errorf("main.go: %w", err)
	}
	cmds := commands.New(cfg, s)
	return cmds, nil
}

func main() {
	parser := flags.NewParser(&options, flags.Default)
	// allow nom to be run without any subcommands
	parser.SubcommandsOptional = true

	// add commands
	parser.AddCommand("add", "Add feed", "Add a new feed", &Add{})
	parser.AddCommand("config", "Show config", "Show configuration", &Config{})
	parser.AddCommand("list", "List feeds", "List all feeds", &List{})
	parser.AddCommand("version", "Show Version", "Display version information", &Version{})
	parser.AddCommand("refresh", "Refresh feeds", "refresh feed(s) without opening TUI", &Refresh{})
	parser.AddCommand("unread", "Count unread", "Get count of unread items", &Unread{})

	// parse the command line arguments
	_, err := parser.Parse()

	// check for help flag
	if err != nil {
		if flagErr, ok := err.(*flags.Error); ok && flagErr.Type != flags.ErrHelp {
			parser.WriteHelp(os.Stdout)
		}

		os.Exit(0)
	}

	// no subcommand or help flag, run the TUI
	if parser.Active == nil {
		cmds, err := getCmds()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		err = cmds.TUI()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		return
	}
}
