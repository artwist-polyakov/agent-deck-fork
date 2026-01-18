package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/asheshgoplani/agent-deck/internal/experiments"
	"github.com/asheshgoplani/agent-deck/internal/session"
)

// handleTry handles the 'try' subcommand for quick experiments
func handleTry(profile string, args []string) {
	fs := flag.NewFlagSet("try", flag.ExitOnError)
	listOnly := fs.Bool("list", false, "List experiments without creating session")
	listShort := fs.Bool("l", false, "List experiments (short)")
	jsonOutput := fs.Bool("json", false, "Output as JSON")
	tool := fs.String("cmd", "", "AI tool to use (defaults to config)")
	toolShort := fs.String("c", "", "AI tool to use (short)")
	noSession := fs.Bool("no-session", false, "Create folder only, don't start session")

	fs.Usage = func() {
		fmt.Println("Usage: agent-deck try <name> [options]")
		fmt.Println()
		fmt.Println("Quick experiment: find or create a dated folder and start a session.")
		fmt.Println()
		fmt.Println("Arguments:")
		fmt.Println("  <name>    Experiment name (fuzzy matched against existing experiments)")
		fmt.Println()
		fmt.Println("Options:")
		fs.PrintDefaults()
		fmt.Println()
		fmt.Println("Examples:")
		fmt.Println("  agent-deck try redis-cache          # Create/find redis-cache experiment")
		fmt.Println("  agent-deck try rds                  # Fuzzy match 'redis-cache'")
		fmt.Println("  agent-deck try --list               # List all experiments")
		fmt.Println("  agent-deck try --list redis         # Fuzzy search experiments")
		fmt.Println("  agent-deck try myproject -c gemini  # Use Gemini instead of Claude")
		fmt.Println("  agent-deck try myproject --no-session  # Just create folder")
		fmt.Println()
		fmt.Println("Config (~/.agent-deck/config.toml):")
		fmt.Println("  [experiments]")
		fmt.Println("  directory = \"~/src/tries\"    # Base directory for experiments")
		fmt.Println("  date_prefix = true           # Add YYYY-MM-DD- prefix")
		fmt.Println("  default_tool = \"claude\"     # Default AI tool")
	}

	// Reorder args: move name to end so flags are parsed correctly
	// Go's flag package stops parsing at first non-flag argument
	// This allows: "try myproject --no-session" to work same as "try --no-session myproject"
	args = reorderArgsForTryCommand(args)

	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	// Get settings
	settings := session.GetExperimentsSettings()

	// Merge flags
	listMode := *listOnly || *listShort
	selectedTool := mergeFlags(*tool, *toolShort)
	if selectedTool == "" {
		selectedTool = settings.DefaultTool
	}

	// Handle list mode
	if listMode {
		handleTryList(settings.Directory, fs.Arg(0), *jsonOutput)
		return
	}

	// Require name for create/find mode
	name := fs.Arg(0)
	if name == "" {
		fs.Usage()
		os.Exit(1)
	}

	// Find or create experiment
	exp, created, err := experiments.FindOrCreate(settings.Directory, name, settings.DatePrefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if *noSession {
		// Just print the path
		if created {
			fmt.Printf("Created: %s\n", exp.Path)
		} else {
			fmt.Printf("Found: %s\n", exp.Path)
		}
		return
	}

	// Create and start session
	storage, err := session.NewStorageWithProfile(profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	instances, groups, err := storage.LoadWithGroups()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Check if session already exists for this path
	for _, inst := range instances {
		if inst.ProjectPath == exp.Path {
			// Session exists - just start it if not running
			if !inst.Exists() {
				if err := inst.Start(); err != nil {
					fmt.Fprintf(os.Stderr, "Error starting session: %v\n", err)
					os.Exit(1)
				}
			}
			fmt.Printf("Session: %s (%s)\n", inst.Title, inst.ID[:8])
			fmt.Printf("Path: %s\n", exp.Path)
			return
		}
	}

	// Create new session
	newInst := session.NewInstanceWithGroup(exp.Name, exp.Path, "experiments")
	newInst.Command = selectedTool
	newInst.Tool = detectTool(selectedTool)

	instances = append(instances, newInst)

	// Rebuild group tree and ensure experiments group exists
	groupTree := session.NewGroupTreeWithGroups(instances, groups)
	groupTree.CreateGroup("experiments")

	if err := storage.SaveWithGroups(instances, groupTree); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Start the session
	if err := newInst.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting session: %v\n", err)
		os.Exit(1)
	}

	action := "Created"
	if !created {
		action = "Found"
	}

	fmt.Printf("%s %s experiment: %s\n", successSymbol, action, exp.Name)
	fmt.Printf("  Path: %s\n", exp.Path)
	fmt.Printf("  Session: %s (%s)\n", newInst.Title, newInst.ID[:8])
	fmt.Printf("  Tool: %s\n", selectedTool)
}

// handleTryList lists experiments with optional fuzzy search
func handleTryList(dir, query string, jsonOutput bool) {
	exps, err := experiments.ListExperiments(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if query != "" {
		exps = experiments.FuzzyFind(exps, query)
	}

	if len(exps) == 0 {
		if query != "" {
			fmt.Printf("No experiments matching %q in %s\n", query, dir)
		} else {
			fmt.Printf("No experiments in %s\n", dir)
		}
		return
	}

	if jsonOutput {
		// JSON output
		fmt.Print("[")
		for i, exp := range exps {
			if i > 0 {
				fmt.Print(",")
			}
			date := ""
			if exp.HasDate {
				date = exp.Date.Format("2006-01-02")
			}
			fmt.Printf(`{"name":%q,"path":%q,"date":%q,"modified":%q}`,
				exp.Name, exp.Path, date, exp.ModTime.Format("2006-01-02 15:04"))
		}
		fmt.Println("]")
		return
	}

	// Table output
	fmt.Printf("Experiments in %s:\n\n", dir)
	fmt.Printf("%-25s %-12s %s\n", "NAME", "DATE", "PATH")
	fmt.Println(strings.Repeat("-", 70))
	for _, exp := range exps {
		date := ""
		if exp.HasDate {
			date = exp.Date.Format("2006-01-02")
		}
		// Truncate path for display
		path := exp.Path
		if len(path) > 30 {
			path = "..." + path[len(path)-27:]
		}
		fmt.Printf("%-25s %-12s %s\n", truncate(exp.Name, 25), date, path)
	}
	fmt.Printf("\nTotal: %d experiments\n", len(exps))
}

// reorderArgsForTryCommand moves the experiment name to the end of args
// so Go's flag package can parse all flags correctly.
// Go's flag package stops parsing at the first non-flag argument,
// so "try myproject --no-session" would fail to parse --no-session without this fix.
// This reorders to "try --no-session myproject" which parses correctly.
func reorderArgsForTryCommand(args []string) []string {
	if len(args) == 0 {
		return args
	}

	// Known flags that take a value (need to skip their values)
	valueFlags := map[string]bool{
		"-c": true, "--cmd": true, "-cmd": true,
	}

	var flags []string
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]

		// Check if it's a flag
		if strings.HasPrefix(arg, "-") {
			flags = append(flags, arg)

			// Check if this flag takes a value (and value is separate)
			// Handle both "-c value" and "-c=value" formats
			if !strings.Contains(arg, "=") && valueFlags[arg] && i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
		} else {
			// Non-flag argument (experiment name)
			positional = append(positional, arg)
		}
	}

	// Return flags first, then positional args
	return append(flags, positional...)
}
