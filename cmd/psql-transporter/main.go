package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jayps/psql-transporter/internal/config"
	"github.com/jayps/psql-transporter/internal/psql"
	"github.com/jayps/psql-transporter/internal/ui"
)

var version = "dev" // overridden by -ldflags "-X main.version=..."

func main() {
	showVersion := flag.Bool("v", false, "Print version and exit")
	flag.BoolVar(showVersion, "version", false, "Print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Println("psql-transporter version:", version)
		os.Exit(0)
	}

	root := &cobra.Command{
		Use:   "psql-transporter",
		Short: "DB export/import helper for Postgres",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgPath, created, err := config.EnsureExists(".")
			if err != nil {
				return err
			}
			if created {
				fmt.Println("Created default config at", cfgPath)
				fmt.Println("Edit it and re-run.")
				return nil
			}

			c, err := config.Load(cfgPath)
			if err != nil {
				return err
			}

			names := make([]string, len(c.Sources))
			for i, s := range c.Sources {
				names[i] = s.Name
			}
			loadFromFileOption := "Load from file"
			namesSrc := make([]string, 0, len(names)+1)
			namesSrc = append(namesSrc, names...)
			namesSrc = append(namesSrc, loadFromFileOption)
			srcSel, err := ui.Select("Select SOURCE:", namesSrc)
			if err != nil {
				return err
			}

			var src *config.Source
			srcIsFile := false
			srcFile := ""
			if srcSel == loadFromFileOption {
				// Source is a dump file
				defPath := filepath.Join(".", "dump.sql")
				filePath, err := ui.InputExistingFile("Enter input dump file path:", defPath)
				if err != nil {
					return err
				}
				srcIsFile = true
				srcFile = filePath
			} else {
				// Find source DB
				for i := range c.Sources {
					if c.Sources[i].Name == srcSel {
						src = &c.Sources[i]
						break
					}
				}
				if src == nil {
					return errors.New("invalid source selection")
				}
			}

			// Build destination options
			dumpToFileOption := "Dump to file"
			namesDst := make([]string, 0, len(names)+1)
			namesDst = append(namesDst, names...)
			if !srcIsFile {
				// Only allow dump-to-file when source is a DB
				namesDst = append(namesDst, dumpToFileOption)
			}
			dstName, err := ui.Select("Select DESTINATION:", namesDst)
			if err != nil {
				return err
			}

			ctx, cancel := psql.DefaultTimeoutCtx()
			defer cancel()

			if !srcIsFile && dstName == dumpToFileOption {
				// DB -> File (export only)
				defPath := filepath.Join(".", "dump.sql")
				filePath, err := ui.Input("Enter output dump file path:", defPath)
				if err != nil {
					return err
				}
				if err := ui.RunSteps([]ui.Step{{
					Title: "Exporting...",
					Run: func() error { return psql.Dump(ctx, toConn(*src), filePath) },
				}}); err != nil {
					return err
				}
				fmt.Println("Dump written to", filePath)
				fmt.Println("All done ✅")
				return nil
			}

			// Destination must be a DB at this point
			var dst *config.Source
			for i := range c.Sources {
				if c.Sources[i].Name == dstName {
					dst = &c.Sources[i]
					break
				}
			}
			if dst == nil {
				return errors.New("invalid destination selection")
			}
			if dst.Protected {
				return fmt.Errorf("destination %q is protected; aborting", dst.Name)
			}

			if !srcIsFile && src.Name == dst.Name {
				return fmt.Errorf("source and destination cannot be the same")
			}

			// Confirm destructive action
			var confirmMsg string
			if srcIsFile {
				confirmMsg = fmt.Sprintf("DESTINATION %q will be WIPED and replaced with contents of %q. Continue?", dst.Name, srcFile)
			} else {
				confirmMsg = fmt.Sprintf("DESTINATION %q will be WIPED and replaced with %q. Continue?", dst.Name, src.Name)
			}
			ok, err := ui.ConfirmDanger(confirmMsg)
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("Aborted.")
				return nil
			}

			if srcIsFile {
				// File -> DB: wipe then import from file
				if err := ui.RunSteps([]ui.Step{
					{Title: "Wiping destination...", Run: func() error { return psql.Wipe(ctx, toConn(*dst)) }},
					{Title: "Importing...", Run: func() error { return psql.Import(ctx, toConn(*dst), srcFile) }},
				}); err != nil {
					return err
				}
				fmt.Println("All done ✅")
				return nil
			}

			// DB -> DB flow (export, wipe, import)
			dumpPath := filepath.Join(".", "dump.sql")
			if err := ui.RunSteps([]ui.Step{
				{Title: "Exporting...", Run: func() error { return psql.Dump(ctx, toConn(*src), dumpPath) }},
				{Title: "Wiping destination...", Run: func() error { return psql.Wipe(ctx, toConn(*dst)) }},
				{Title: "Importing...", Run: func() error { return psql.Import(ctx, toConn(*dst), dumpPath) }},
			}); err != nil {
				return err
			}

			fmt.Println("All done ✅")
			return nil
		},
	}

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func toConn(s config.Source) psql.Conn {
	return psql.Conn{
		Host: s.Host, Port: s.Port,
		User: s.User, Password: s.Password,
		DBName: s.DBName, SSLMode: s.SSLMode,
	}
}
