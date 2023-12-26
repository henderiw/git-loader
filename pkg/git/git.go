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
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	configv1alpha1 "github.com/henderiw/git-loader/apis/config/v1alpha1"
	"github.com/henderiw/git-loader/pkg/auth"
	"github.com/henderiw/logger/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

var tracer = otel.Tracer("git")

type GitRepository interface {
	List(ctx context.Context, ref string, listFn ListFunc) error
	Commit(ctx context.Context, ref, packageName, workspaceName, revision string, resources map[string]string) error
	Push(ctx context.Context, ref string) error
}

type gitRepository struct {
	url                string
	secret             string  // Secret containing Credentials
	ref                RefName // The main branch from repository registration (defaults to 'main' if unspecified)
	directory          string
	repo               *git.Repository
	credentialResolver auth.CredentialResolver
	userInfoProvider   auth.UserInfoProvider

	// credential contains the information needed to authenticate against
	// a git repository.
	credential auth.Credential

	mu sync.Mutex
}

type Options struct {
	CredentialResolver auth.CredentialResolver
	UserInfoProvider   auth.UserInfoProvider
}

func OpenRepository(ctx context.Context, root string, repoCfg *configv1alpha1.GitRepository, opts *Options) (GitRepository, error) {
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
		url:                repoCfg.URL,
		secret:             repoCfg.Credentials,
		ref:                ref,
		directory:          strings.Trim(repoCfg.Directory, "/"),
		repo:               repo,
		credentialResolver: opts.CredentialResolver,
		userInfoProvider:   opts.UserInfoProvider,
	}

	if err := repository.fetchRemoteRepository(ctx); err != nil {
		return nil, err
	}

	if _, err := repository.verifyRef(ctx, ref); err != nil {
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
	/*
		if r.secret == "" {
			return nil, nil
		}
	*/

	if r.credential == nil || !r.credential.Valid() || forceRefresh {
		if cred, err := r.credentialResolver.ResolveCredential(ctx, "", ""); err != nil {
			return nil, fmt.Errorf("failed to obtain credential from secret %s/%s: %w", "", "", err)
		} else {
			r.credential = cred
		}
	}

	return r.credential.ToAuthMethod(), nil
}

func (r *gitRepository) getCommit(ctx context.Context, ref RefName) (*object.Commit, error) {
	var err error
	var commit *object.Commit

	// verify if the ref is in the repository and if the ref is a branch or a tag
	// since the commit hash lookup is different depending if it is a branch or a tag
	branch, err := r.verifyRef(ctx, ref)
	if err != nil {
		return nil, err
	}
	if branch {
		commit, err = r.getCommitFromBranch(ctx, plumbing.ReferenceName(branchPrefixInLocalRepo+string(ref)))
		if err != nil {
			return nil, err
		}
	} else {
		commit, err = r.getCommitFromTag(ctx, plumbing.ReferenceName(tagsPrefixInLocalRepo+string(ref)))
		if err != nil {
			return nil, err
		}
	}
	return commit, nil
}

// Verifies reference in the repository and returns true if it is a branch and false
// if it is a tag or an error if not found
func (r *gitRepository) verifyRef(ctx context.Context, ref RefName) (bool, error) {
	if _, err := r.repo.Reference(ref.RefInLocal(), false); err == nil {
		return true, nil
	}
	if _, err := r.repo.Reference(ref.TagInLocal(), false); err == nil {
		return false, nil
	}
	return false, fmt.Errorf("no branches/tags found for this ref %q", ref)
}

func (r *gitRepository) getCommitFromBranch(ctx context.Context, refname plumbing.ReferenceName) (*object.Commit, error) {
	ref, err := r.repo.Reference(refname, false)
	if err != nil {
		return nil, err
	}
	return r.repo.CommitObject(ref.Hash())
}

func (r *gitRepository) getCommitFromTag(ctx context.Context, refname plumbing.ReferenceName) (*object.Commit, error) {
	ref, err := r.repo.Reference(refname, false)
	if err != nil {
		return nil, err
	}
	tag, err := r.repo.TagObject(ref.Hash())
	if err != nil {
		return nil, err
	}

	return tag.Commit()
}

func (r *gitRepository) getRootTree(ctx context.Context, commit *object.Commit) (*object.Tree, error) {
	//log := log.FromContext(ctx)
	rootTree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("cannot resolve commit %v to tree (corrupted repository?): %w", commit.Hash, err)
	}

	if r.directory != "" {
		tree, err := rootTree.Tree(r.directory)
		if err != nil {
			return nil, err
		}
		rootTree = tree
	}
	return rootTree, nil
}

// pushes the local reference to the remote repository
func (r *gitRepository) pushAndCleanup(ctx context.Context, ph *pushRefSpecBuilder) error {
	specs, require, err := ph.BuildRefSpecs()
	if err != nil {
		return err
	}

	if err := r.doGitWithAuth(ctx, func(auth transport.AuthMethod) error {
		return r.repo.Push(&git.PushOptions{
			RemoteName:        OriginName, // origin
			RefSpecs:          specs, // e.g. [d48aaa68deca311768be2bb5dd0cd97b8da13971:refs/heads/test-package/test-workspace]
			Auth:              auth,
			RequireRemoteRefs: require, // empty for push
			Force: true,
		})
	}); err != nil {
		return err
	}
	return nil
}