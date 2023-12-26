package token

import (
	"context"

	"github.com/henderiw/git-loader/pkg/auth"
)

type Resolver interface {
	Resolve(ctx context.Context) (auth.Credential, bool, error)
}
