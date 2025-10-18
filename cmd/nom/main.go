package main

import (
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"

	"github.com/guyfedwards/nom/v2/internal/commands"
	"github.com/guyfedwards/nom/v2/internal/config"
	"github.com/guyfedwards/nom/v2/internal/store"
	"github.com/guyfedwards/nom/v2/internal/version"
)

type Options struct {
	Verbose      bool     `short:"v" long:"verbose" description:"Show verbose logging"`
	Pager        string   `short:"p" long:"pager" description:"Pager to use for longer output. Set to false for no pager"`
	ConfigPath   string   `short:"c" long:"config-path" description:"Location of config.yml" env:"NOM_CONFIG_FILE"`
	ConfigDir    string   `short:"d" long:"config-dir" description:"Where to find config files"`
	ConfigName   string   `short:"N" long:"config-name" description:"Name of a config file in config dir"`
	PreviewFeeds []string `short:"f" long:"feed" description:"Feed(s) URL(s) for preview"`
	Create       bool     `long:"create" description:"Create config file if it doesn't exist"`
}

var (
	options Options
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
	fmt.Print(version.BuildVersion)
	if version.BuildRef != "" {
		fmt.Printf(" (%s)", version.BuildRef)
	}
	if version.BuildDate != "" {
		fmt.Printf(" on %s", version.BuildDate)
	}
	fmt.Println()
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

type Import struct {
	Positional struct {
		Source string `positional-arg-name:"SOURCE" required:"yes" description:"Source OPML data. Can be either a file path or a URL"`
	} `positional-args:"yes"`
}

func (r *Import) Execute(args []string) error {
	cmds, err := getCmds()
	if err != nil {
		return err
	}

	err = cmds.ImportFeeds(r.Positional.Source)
	if err != nil {
		return err
	}
	return nil
}

func getCmds() (*commands.Commands, error) {
	runtime, err := config.New().
		WithConfigPath(options.ConfigPath).
		WithConfigDir(options.ConfigDir).
		WithConfigName(options.ConfigName).
		WithPreviewFeeds(options.PreviewFeeds).
		WithVersion(version.BuildVersion).
		WithCreate(options.Create).
		Load()
	if err != nil {
		return nil, err
	}

	// Apply command line options that should override
	// config file options.
	runtime = runtime.
		WithPager(options.Pager)

	var s store.Store
	if runtime.IsPreviewMode() {
		s, err = store.NewInMemorySQLiteStore()
	} else {
		s, err = store.NewSQLiteStore(runtime.ConfigDir, runtime.Config.Database)
	}
	if err != nil {
		return nil, fmt.Errorf("main.go: %w", err)
	}
	cmds := commands.New(runtime, s)
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
	parser.AddCommand("import", "Import feeds", "Import feeds from an OMPL file", &Import{})

	// parse the command line arguments
	_, err := parser.Parse()
	// check for help flag
	if err != nil {
		if flagErr, ok := err.(*flags.Error); ok && flagErr.Type == flags.ErrHelp {
			os.Exit(0)
		}

		parser.WriteHelp(os.Stdout)
		os.Exit(2)
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
	}
}
