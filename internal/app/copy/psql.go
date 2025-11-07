package copy

import (
	"context"
	"fmt"
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
	args := append(src.baseArgs(),
		"--no-owner", "--no-privileges", "-F", "p", "-f", outFile,
	)
	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Env = src.env()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Wipe(ctx context.Context, dst Conn) error {
	args := append(dst.baseArgs(),
		"-c", `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`,
	)
	cmd := exec.CommandContext(ctx, "psql", args...)
	cmd.Env = dst.env()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Import(ctx context.Context, dst Conn, file string) error {
	args := append(dst.baseArgs(), "-f", file)
	cmd := exec.CommandContext(ctx, "psql", args...)
	cmd.Env = dst.env()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func DefaultTimeoutCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Minute)
}
