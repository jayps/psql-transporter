package main

import (
	"errors"
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
			srcName, err := ui.Select("Select SOURCE database:", names)
			if err != nil {
				return err
			}
			dstName, err := ui.Select("Select DESTINATION database:", names)
			if err != nil {
				return err
			}

			var src, dst *config.Source
			for i := range c.Sources {
				if c.Sources[i].Name == srcName {
					src = &c.Sources[i]
				}
				if c.Sources[i].Name == dstName {
					dst = &c.Sources[i]
				}
			}
			if src == nil || dst == nil {
				return errors.New("invalid selection")
			}
			if dst.Protected {
				return fmt.Errorf("destination %q is protected; aborting", dst.Name)
			}
			if src.Name == dst.Name {
				return fmt.Errorf("source and destination cannot be the same")
			}

			ok, err := ui.ConfirmDanger(fmt.Sprintf(
				"DESTINATION %q will be WIPED and replaced with %q. Continue?",
				dst.Name, src.Name,
			))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("Aborted.")
				return nil
			}

			dumpPath := filepath.Join(".", "dump.sql")
			ctx, cancel := psql.DefaultTimeoutCtx()
			defer cancel()

			err = ui.RunSteps([]ui.Step{
				{
					Title: "Exporting...",
					Run:   func() error { return psql.Dump(ctx, toConn(*src), dumpPath) },
				},
				{
					Title: "Wiping destination...",
					Run:   func() error { return psql.Wipe(ctx, toConn(*dst)) },
				},
				{
					Title: "Importing...",
					Run:   func() error { return psql.Import(ctx, toConn(*dst), dumpPath) },
				},
			})
			if err != nil {
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
