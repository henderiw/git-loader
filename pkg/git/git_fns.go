package git

import (
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/henderiw/logger/log"
	"go.opentelemetry.io/otel/trace"
)

type ListFunc func(ctx context.Context, tree *object.Tree) error

func (r *gitRepository) List(ctx context.Context, ref string, listFn ListFunc) error {
	ctx, span := tracer.Start(ctx, "gitRepository::List", trace.WithAttributes())
	defer span.End()
	r.mu.Lock()
	defer r.mu.Unlock()

	log := log.FromContext(ctx)
	// getCommit
	commit, err := r.getCommit(ctx, RefName(ref))
	if err != nil {
		return err
	}
	tree, err := r.getRootTree(ctx, commit)
	if err != nil {
		if err == object.ErrDirectoryNotFound {
			log.Info("could not find directory prefix in commit", "path", r.directory, "commit", commit.Hash.String())
			return nil
		} else {
			return err
		}
	}
	if listFn != nil {
		return listFn(ctx, tree)
	}
	return nil
}

func (r *gitRepository) Commit(ctx context.Context, ref, packageName, workspaceName, revision string, resources map[string]string) error {
	ctx, span := tracer.Start(ctx, "gitRepository::Create", trace.WithAttributes())
	defer span.End()
	r.mu.Lock()
	defer r.mu.Unlock()

	// check if the new ref already exists
	var parentCommit *object.Commit
	if _, err := r.repo.Reference(plumbing.ReferenceName(ref), false); err != nil {
		fmt.Println("Create from base")
		// create -> ref does no exist
		// get the main ref of the repository -> typically main
		parentCommit, err = r.getCommit(ctx, r.ref)
		if err != nil {
			// We dont support empty repositories
			return err
		}
		localRef := plumbing.NewHashReference(plumbing.ReferenceName(ref), parentCommit.Hash)
		if err := r.repo.Storer.SetReference(localRef); err != nil {
			return err
		}

	} else {
		fmt.Println("Update reference")
		// update -> ref already exists
		parentCommit, err = r.getCommitFromBranch(ctx, plumbing.ReferenceName(ref))
		if err != nil {
			// Strange
			return err
		}
	}
	packagePath := filepath.Join(r.directory, packageName)
	ch, err := newCommitHelper(ctx, r, parentCommit.Hash, packagePath, plumbing.ZeroHash)
	if err != nil {
		return nil
	}

	for k, v := range resources {
		ch.storeFile(path.Join(packagePath, k), v)
	}

	annotation := &gitAnnotation{
		PackagePath:   packagePath,
		WorkspaceName: workspaceName,
		Revision:      revision,
		//Task:          change,
	}
	message := "Intermediate commit"
	/*
		if change != nil {
			message += fmt.Sprintf(": %s", change.Type)
			draft.tasks = append(draft.tasks, *change)
		}
	*/
	message += "\n"
	message, err = AnnotateCommitMessage(message, annotation)
	if err != nil {
		return err
	}

	commitHash, packageTree, err := ch.commit(ctx, message, packagePath)
	if err != nil {
		return fmt.Errorf("failed to commit package: %w", err)
	}
	fmt.Println("commitHash", commitHash)
	fmt.Println("packageTree", packageTree)

	localRef := plumbing.NewHashReference(plumbing.ReferenceName(ref), commitHash)
	fmt.Println("localRef", localRef)
	if err := r.repo.Storer.SetReference(localRef); err != nil {
		return err
	}
	fmt.Println("Commit", nil)

	return nil
}

func (r *gitRepository) Push(ctx context.Context, ref string) error {
	ctx, span := tracer.Start(ctx, "gitRepository::Push", trace.WithAttributes())
	defer span.End()
	r.mu.Lock()
	defer r.mu.Unlock()

	refSpecs := newPushRefSpecBuilder()

	// Find the local reference -> to find out if it exists
	localref, err := r.repo.Reference(plumbing.ReferenceName(ref), false)
	if err != nil {
		return err
	}

	// Get the commit hash related to the reference
	commit, err := r.getCommitFromBranch(ctx, plumbing.ReferenceName(ref))
	if err != nil {
		return err
	}

	// build the refs to push to the remote reference
	refSpecs.AddRefToPush(localref.Name(), commit.Hash)
	if err := r.pushAndCleanup(ctx, refSpecs); err != nil {
		if !errors.Is(err, git.NoErrAlreadyUpToDate) {
			return err
		}
	}

	return nil
}
