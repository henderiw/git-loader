package secret

import (
	"context"

	"github.com/henderiw/git-loader/pkg/auth"
	corev1 "k8s.io/api/core/v1"
)

type Resolver interface {
	Resolve(ctx context.Context, secret corev1.Secret) (auth.Credential, bool, error)
}
