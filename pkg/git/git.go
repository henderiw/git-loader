package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	configv1alpha1 "github.com/henderiw/git-loader/apis/config/v1alpha1"
	"github.com/henderiw/logger/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("git")

type GitRepository interface {
	List(ctx context.Context, ref string, listFn ListFunc) error
}

type gitRepository struct {
	url          string
	secret       string  // Secret containing Credentials
	ref          RefName // The main branch from repository registration (defaults to 'main' if unspecified)
	directory    string
	repo         *git.Repository
	branchNotTag bool // is either a branch or a tag -> dynamically discovered based on the tag we get from the repoCfg

	mu sync.Mutex
}

func OpenRepository(ctx context.Context, root string, repoCfg *configv1alpha1.GitRepository) (GitRepository, error) {
	ctx, span := tracer.Start(ctx, "OpenRepository", trace.WithAttributes())
	defer span.End()

	replace := strings.NewReplacer("/", "-", ":", "-")
	dir := filepath.Join(root, replace.Replace(repoCfg.URL))

	// Cleanup the directory in case initialization fails.
	cleanup := dir
	defer func() {
		if cleanup != "" {
			os.RemoveAll(cleanup)
		}
	}()

	var repo *git.Repository

	// check if the directory exists (<init-dir>/<git>/<repo-url w/ replaced / and :>)
	if fi, err := os.Stat(dir); err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		r, err := initEmptyRepository(dir)
		if err != nil {
			return nil, fmt.Errorf("error cloning git repository %q: %w", repoCfg.URL, err)
		}

		repo = r

	} else if !fi.IsDir() {
		// file exists but is not a directory -> corruption
		return nil, fmt.Errorf("cannot clone git repository %q: %w", repoCfg.URL, err)
	} else {
		// director that exists
		cleanup = "" // do no cleanup
		r, err := openRepository(dir)
		if err != nil {
			return nil, err
		}
		repo = r
	}

	// Create Remote
	if err := initializeOrigin(repo, repoCfg.URL); err != nil {
		return nil, fmt.Errorf("error cloning git repository %q, cannot create remote: %v", repoCfg.URL, err)
	}

	ref := MainBranch
	if repoCfg.Ref != "" {
		ref = RefName(repoCfg.Ref)
	}

	repository := &gitRepository{
		url:       repoCfg.URL,
		secret:    repoCfg.Credentials,
		ref:       ref,
		directory: strings.Trim(repoCfg.Directory, "/"),
		repo:      repo,
		//credentialResolver: opts.CredentialResolver,
		//userInfoProvider:   opts.UserInfoProvider,
	}

	if err := repository.fetchRemoteRepository(ctx); err != nil {
		return nil, err
	}

	if err := repository.verifyRepository(ctx); err != nil {
		return nil, err
	}

	cleanup = "" // success we are good to go w/o removing the directory

	return repository, nil
}

func (r *gitRepository) fetchRemoteRepository(ctx context.Context) error {
	ctx, span := tracer.Start(ctx, "gitRepository::fetchRemoteRepository", trace.WithAttributes())
	defer span.End()

	// Fetch
	switch err := r.doGitWithAuth(ctx, func(auth transport.AuthMethod) error {
		return r.repo.Fetch(&git.FetchOptions{
			RemoteName: OriginName,
			Auth:       auth,
		})
	}); err {
	case nil: // OK
	case git.NoErrAlreadyUpToDate:
	case transport.ErrEmptyRemoteRepository:

	default:
		return fmt.Errorf("cannot fetch repository %q: %w", r.url, err)
	}

	return nil
}

// Verifies repository. Repository must be fetched already.
func (r *gitRepository) verifyRepository(ctx context.Context) error {
	if _, err := r.repo.Reference(r.ref.RefInLocal(), false); err == nil {
		r.branchNotTag = true
		return nil
	}
	if _, err := r.repo.Reference(r.ref.TagInLocal(), false); err == nil {
		r.branchNotTag = false // this is a tag
		return nil
	}
	return fmt.Errorf("no branches/tags found for this ref %q", r.ref)
}

// doGitWithAuth fetches auth information for git and provides it
// to the provided function which performs the operation against a git repo.
func (r *gitRepository) doGitWithAuth(ctx context.Context, op func(transport.AuthMethod) error) error {
	log := log.FromContext(ctx)
	auth, err := r.getAuthMethod(ctx, false)
	if err != nil {
		return fmt.Errorf("failed to obtain git credentials: %w", err)
	}
	err = op(auth)
	if err != nil {
		if !errors.Is(err, transport.ErrAuthenticationRequired) {
			return err
		}
		log.Info("Authentication failed. Trying to refresh credentials")
		// TODO: Consider having some kind of backoff here.
		auth, err := r.getAuthMethod(ctx, true)
		if err != nil {
			return fmt.Errorf("failed to obtain git credentials: %w", err)
		}
		return op(auth)
	}
	return nil
}

// getAuthMethod fetches the credentials for authenticating to git. It caches the
// credentials between calls and refresh credentials when the tokens have expired.
func (r *gitRepository) getAuthMethod(ctx context.Context, forceRefresh bool) (transport.AuthMethod, error) {
	// If no secret is provided, we try without any auth.
	if r.secret == "" {
		return nil, nil
	}

	/*
		if r.credential == nil || !r.credential.Valid() || forceRefresh {
			if cred, err := r.credentialResolver.ResolveCredential(ctx, r.namespace, r.secret); err != nil {
				return nil, fmt.Errorf("failed to obtain credential from secret %s/%s: %w", r.namespace, r.secret, err)
			} else {
				r.credential = cred
			}
		}

		return r.credential.ToAuthMethod(), nil
	*/
	return nil, nil
}
