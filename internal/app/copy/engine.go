package copy

import (
	"context"
	"os/exec"
)

type Runner interface {
	Run(ctx context.Context, srcConn, dstConn string) error
}

// PGDumpEngine keeps your current behavior: shell out to pg_dump/psql.
type PGDumpEngine struct {
	LookPath func(file string) (string, error) // default: exec.LookPath
	Command  func(name string, arg ...string) *exec.Cmd
}

func NewPGDumpEngine() *PGDumpEngine {
	return &PGDumpEngine{
		LookPath: exec.LookPath,
		Command:  exec.Command,
	}
}

func (e *PGDumpEngine) Run(ctx context.Context, srcConn, dstConn string) error {
	// This is a placeholder where you paste/move your existing logic
	// that calls pg_dump and psql, preserving exact flags/behavior.
	// Example skeleton (replace with your current implementation):

	// 1) pg_dump
	// 2) psql restore

	// Return first error encountered to preserve current UX semantics.
	return nil
}
