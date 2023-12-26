package secret

import (
	"context"
	"fmt"

	"github.com/henderiw/git-loader/pkg/auth"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// Values for scret types supported by porch.
	BasicAuthType = corev1.SecretTypeBasicAuth
)

func NewCredentialResolver(client client.Reader, resolverChain []Resolver) auth.CredentialResolver {
	return &secretResolver{
		client:        client,
		resolverChain: resolverChain,
	}
}

type secretResolver struct {
	resolverChain []Resolver
	client        client.Reader
}

var _ auth.CredentialResolver = &secretResolver{}

func (r *secretResolver) ResolveCredential(ctx context.Context, namespace, name string) (auth.Credential, error) {
	var secret corev1.Secret
	if err := r.client.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, &secret); err != nil {
		return nil, fmt.Errorf("cannot resolve credentials in a secret %s/%s: %w", namespace, name, err)
	}
	for _, resolver := range r.resolverChain {
		cred, found, err := resolver.Resolve(ctx, secret)
		if err != nil {
			return nil, fmt.Errorf("error resolving credential: %w", err)
		}
		if found {
			return cred, nil
		}
	}
	return nil, &NoMatchingResolverError{
		Type: string(secret.Type),
	}
}

type NoMatchingResolverError struct {
	Type string
}

func (e *NoMatchingResolverError) Error() string {
	return fmt.Sprintf("no resolver for secret with type %s", e.Type)
}

func (e *NoMatchingResolverError) Is(err error) bool {
	nmre, ok := err.(*NoMatchingResolverError)
	if !ok {
		return false
	}
	return nmre.Type == e.Type
}