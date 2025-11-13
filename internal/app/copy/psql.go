package copy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

type Conn struct {
	Host, User, Password, DBName, SSLMode string
	Port                                  int
}

func (c Conn) env() []string {
	env := os.Environ()
	env = append(env, "PGPASSWORD="+c.Password)
	if c.SSLMode != "" {
		env = append(env, "PGSSLMODE="+c.SSLMode)
	}
	return env
}

func (c Conn) baseArgs() []string {
	return []string{
		"-h", c.Host,
		"-p", fmt.Sprintf("%d", c.Port),
		"-U", c.User,
		"-d", c.DBName,
	}
}

func Dump(ctx context.Context, src Conn, outFile string) error {
	// Backwards-compatible: call DumpWithProgress with no-op progress to avoid stdout flicker.
	return DumpWithProgress(ctx, src, outFile, nil)
}

// DumpWithProgress runs pg_dump and periodically reports the current output file size
// via the onSize callback. If onSize is nil, progress is suppressed.
func DumpWithProgress(ctx context.Context, src Conn, outFile string, onSize func(int64)) error {
	args := append(src.baseArgs(),
		"--no-owner", "--no-privileges", "-F", "p", "-f", outFile,
	)
	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Env = src.env()
	// Keep pg_dump quiet; we'll manage any UI externally.
	cmd.Stdout = io.Discard
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	// Poll file size while dump runs.
	ticker := time.NewTicker(500 * time.Millisecond)
	quit := make(chan struct{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case <-ticker.C:
				if onSize != nil {
					if fi, err := os.Stat(outFile); err == nil {
						onSize(fi.Size())
					}
				}
			case <-quit:
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	err := cmd.Wait()
	ticker.Stop()
	close(quit)
	<-done
	if err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("pg_dump failed: %v\n%s", err, stderr.String())
		}
		return err
	}
	return nil
}

func Wipe(ctx context.Context, dst Conn) error {
	args := append(dst.baseArgs(),
		"-c", `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`,
	)
	cmd := exec.CommandContext(ctx, "psql", args...)
	cmd.Env = dst.env()
	// Hide psql output during wipe as well
	cmd.Stdout = io.Discard
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("psql wipe failed: %v\n%s", err, stderr.String())
		}
		return err
	}
	return nil
}

func Import(ctx context.Context, dst Conn, file string) error {
	args := append(dst.baseArgs(), "-f", file)
	cmd := exec.CommandContext(ctx, "psql", args...)
	cmd.Env = dst.env()
	// Hide noisy psql output during import
	cmd.Stdout = io.Discard
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return fmt.Errorf("psql import failed: %v\n%s", err, stderr.String())
		}
		return err
	}
	return nil
}

func DefaultTimeoutCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Minute)
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
	// i indicates how many times we've divided by 1024 so far (0 => KiB, 1 => MiB, 2 => GiB, ...)
	prefix := []string{"KiB", "MiB", "GiB", "TiB", "PiB", "EiB"}
	val := d / float64(int64(unit)<<(10*i))
	if i >= len(prefix) {
		// very large, fall back to bytes
		return fmt.Sprintf("%d B", b)
	}
	return fmt.Sprintf("%.1f %s", val, prefix[i])
}
