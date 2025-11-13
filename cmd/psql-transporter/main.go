package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pterm/pterm"
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

			// Build destination options (omit protected databases and the same as source)
			dumpToFileOption := "Dump to file"
			namesDst := make([]string, 0, len(names)+1)
			for _, s := range c.Sources {
				// skip protected databases for destination choices
				if s.Protected {
					continue
				}
				// if source is a DB, do not allow selecting the same DB as destination
				if !srcIsFile && src != nil && s.Name == src.Name {
					continue
				}
				namesDst = append(namesDst, s.Name)
			}
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
				// Use a spinner and update its text with the file size as the dump progresses
				spinner, _ := pterm.DefaultSpinner.Start("Exporting...")
				err = psql.DumpWithProgress(ctx, toConn(*src), filePath, func(sz int64) {
					spinner.UpdateText(fmt.Sprintf("Exporting... (%s)", humanSize(sz)))
				})
				if err != nil {
					spinner.Fail(fmt.Sprintf("Export failed: %v", err))
					return err
				}
				spinner.Success("Export completed")
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
				// File -> DB: wipe then import from file with progress
				if err := ui.RunSteps([]ui.Step{
					{Title: "Wiping destination...", Run: func() error { return psql.Wipe(ctx, toConn(*dst)) }},
				}); err != nil {
					return err
				}
				spinner, _ := pterm.DefaultSpinner.Start("Importing...")
				err := psql.ImportWithProgress(ctx, toConn(*dst), srcFile, func(done, total int64) {
					var text string
					if total > 0 {
						pct := float64(done) / float64(total) * 100
						text = fmt.Sprintf("Importing... (%.1f%%)", pct)
					} else {
						text = fmt.Sprintf("Importing... (%s)", humanSize(done))
					}
					spinner.UpdateText(text)
				})
				if err != nil {
					spinner.Fail(fmt.Sprintf("Import failed: %v", err))
					return err
				}
				spinner.Success("Import completed")
				fmt.Println("All done ✅")
				return nil
			}

			// DB -> DB flow (export, wipe, import)
			dumpPath := filepath.Join(".", "dump.sql")
			// Custom export step with progress shown in the spinner text
			spinner, _ := pterm.DefaultSpinner.Start("Exporting...")
			err = psql.DumpWithProgress(ctx, toConn(*src), dumpPath, func(sz int64) {
				spinner.UpdateText(fmt.Sprintf("Exporting... (%s)", humanSize(sz)))
			})
			if err != nil {
				spinner.Fail(fmt.Sprintf("Export failed: %v", err))
				return err
			}
			spinner.Success("Export completed")
			if err := ui.RunSteps([]ui.Step{
				{Title: "Wiping destination...", Run: func() error { return psql.Wipe(ctx, toConn(*dst)) }},
			}); err != nil {
				return err
			}
			impSpinner, _ := pterm.DefaultSpinner.Start("Importing...")
			err = psql.ImportWithProgress(ctx, toConn(*dst), dumpPath, func(done, total int64) {
				var text string
				if total > 0 {
					pct := float64(done) / float64(total) * 100
					text = fmt.Sprintf("Importing... (%.1f%%)", pct)
				} else {
					text = fmt.Sprintf("Importing... (%s)", humanSize(done))
				}
				impSpinner.UpdateText(text)
			})
			if err != nil {
				impSpinner.Fail(fmt.Sprintf("Import failed: %v", err))
				return err
			}
			impSpinner.Success("Import completed")

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

// humanSize returns a human-friendly file size using binary units.
func humanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	d := float64(b)
	var i int
	for n := b / unit; n >= unit; n /= unit {
		i++
	}
	prefix := []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	val := d / float64(int64(unit)<<(10*i))
	if i >= len(prefix) {
		return fmt.Sprintf("%d B", b)
	}
	return fmt.Sprintf("%.1f %s", val, prefix[i])
}
