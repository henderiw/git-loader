package token

import (
	"context"
	"fmt"

	"github.com/henderiw/git-loader/pkg/auth"
)

func NewCredentialResolver(resolverChain []Resolver) auth.CredentialResolver {
	return &tokenResolver{
		resolverChain: resolverChain,
	}
}

type tokenResolver struct {
	resolverChain []Resolver
}

func (r *tokenResolver) ResolveCredential(ctx context.Context, namespace, name string) (auth.Credential, error) {
	for _, resolver := range r.resolverChain {
		cred, found, err := resolver.Resolve(ctx)
		if err != nil {
			return nil, fmt.Errorf("error resolving credential: %w", err)
		}
		if found {
			return cred, nil
		}
	}
	return nil, &NoMatchingResolverError{}
}

type NoMatchingResolverError struct {
}

func (e *NoMatchingResolverError) Error() string {
	return "no resolver"
}

func (e *NoMatchingResolverError) Is(err error) bool {
	_, ok := err.(*NoMatchingResolverError)
	return ok
}
