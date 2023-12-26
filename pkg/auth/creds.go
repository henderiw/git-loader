package auth

import (
	"context"

	"github.com/go-git/go-git/v5/plumbing/transport"
)

type CredentialResolver interface {
	ResolveCredential(ctx context.Context, namespace, name string) (Credential, error)
}

type Credential interface {
	Valid() bool
	ToAuthMethod() transport.AuthMethod
}
