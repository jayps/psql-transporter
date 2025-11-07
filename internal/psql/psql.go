package psql

import (
	"context"
	appcopy "github.com/jayps/psql-transporter/internal/app/copy"
)

type (
	Conn = appcopy.Conn
)

func Dump(ctx context.Context, src Conn, outFile string) error {
	return appcopy.Dump(ctx, src, outFile)
}
func Wipe(ctx context.Context, dst Conn) error                 { return appcopy.Wipe(ctx, dst) }
func Import(ctx context.Context, dst Conn, file string) error  { return appcopy.Import(ctx, dst, file) }
func DefaultTimeoutCtx() (context.Context, context.CancelFunc) { return appcopy.DefaultTimeoutCtx() }
