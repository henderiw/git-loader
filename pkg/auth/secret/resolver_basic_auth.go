package secret

import (
	"context"

	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/henderiw/git-loader/pkg/auth"
	corev1 "k8s.io/api/core/v1"
)

func NewBasicAuthResolver() Resolver {
	return &BasicAuthResolver{}
}

var _ Resolver = &BasicAuthResolver{}

type BasicAuthResolver struct{}

func (b *BasicAuthResolver) Resolve(_ context.Context, secret corev1.Secret) (auth.Credential, bool, error) {
	if secret.Type != BasicAuthType {
		return nil, false, nil
	}

	return &BasicAuthCredential{
		Username: string(secret.Data["username"]),
		Password: string(secret.Data["password"]),
	}, true, nil
}

type BasicAuthCredential struct {
	Username string
	Password string
}

var _ auth.Credential = &BasicAuthCredential{}

func (b *BasicAuthCredential) Valid() bool {
	return true
}

func (b *BasicAuthCredential) ToAuthMethod() transport.AuthMethod {
	return &http.BasicAuth{
		Username: string(b.Username),
		Password: string(b.Password),
	}
}
