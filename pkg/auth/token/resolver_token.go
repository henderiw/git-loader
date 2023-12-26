package token

import (
	"context"
	"os"

	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/henderiw/git-loader/pkg/auth"
)

func NewTokenResolver() Resolver {
	return &TokenResolver{}
}

var _ Resolver = &TokenResolver{}

type TokenResolver struct{}

func (b *TokenResolver) Resolve(_ context.Context) (auth.Credential, bool, error) {
	return &TokenCredential{
		Username: os.Getenv("GITHUB_USERNAME"),
		Password: os.Getenv("GITHUB_PASSWORD"),
	}, true, nil
}

type TokenCredential struct {
	Username string
	Password string
}

var _ auth.Credential = &TokenCredential{}

func (r *TokenCredential) Valid() bool {
	return true
}

func (r *TokenCredential) ToAuthMethod() transport.AuthMethod {
	return &http.BasicAuth{
		Username: r.Username,
		Password: r.Password,
	}
}
