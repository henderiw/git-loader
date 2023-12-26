package git

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/object"
)

const (
	commitSignatureName  = "henderiw"
	commitSignatureEmail = "wim.henderickx@gmail.com"
)

type commitHelper struct {
	repository *gitRepository

	// parentCommitHash holds the parent commit, or nil if this is the first commit.
	parentCommitHash plumbing.Hash

	// trees holds a map of all the tree objects we are writing to.
	// We reuse the existing object.Tree structures.
	// When a tree is dirty, we set the hash as plumbing.ZeroHash.
	trees map[string]*object.Tree
}

func newCommitHelper(ctx context.Context, repo *gitRepository, parentCommitHash plumbing.Hash, packagePath string, packageTree plumbing.Hash) (*commitHelper, error) {
	ch := &commitHelper{
		repository:       repo,
		parentCommitHash: parentCommitHash,
	}
	var rootTree *object.Tree
	if parentCommitHash.IsZero() {
		// No parent commit, start with an empty tree
		rootTree = &object.Tree{}
	} else {
		parentCommit, err := ch.repository.repo.CommitObject(parentCommitHash)
		if err != nil {
			return nil, fmt.Errorf("cannot resolve parent commit hash %s to commit: %w", parentCommitHash, err)
		}
		t, err := parentCommit.Tree()
		if err != nil {
			return nil, fmt.Errorf("cannot resolve parent commit's (%s) tree (%s) to tree object: %w", parentCommit.Hash, parentCommit.TreeHash, err)
		}
		rootTree = t
	}
	fmt.Println("newCommitHelper: rootTree", rootTree)
	fmt.Println("newCommitHelper: packagePath", packagePath)
	fmt.Println("newCommitHelper: packageTree", packageTree)
	if err := ch.initializeTrees(ctx, rootTree, packagePath, packageTree); err != nil {
		return nil, err
	}

	for key, val := range ch.trees {
		fmt.Println("key:", key, "val:", val)
	}

	return ch, nil
}

// initializeTrees initializes the tree context in the commitHelper.
// It initialized the ancestor trees of the package.

func (r *commitHelper) initializeTrees(ctx context.Context, rootTree *object.Tree, packagePath string, packageTreeHash plumbing.Hash) error {
	r.trees = map[string]*object.Tree{
		"": rootTree,
	}
	parts := strings.Split(packagePath, "/")
	if len(parts) == 0 {
		// empty package path is invalid
		return fmt.Errorf("invalid package path: %q", packagePath)
	}
	fmt.Println("initializeTrees: parts", parts)

	// Load all ancestor trees
	parent := rootTree
	for i, max := 0, len(parts)-1; i < max; i++ {
		name := parts[i]
		path := strings.Join(parts[0:i+1], "/")

		var current *object.Tree
		switch existing := findTreeEntry(parent, name); {
		case existing == nil:
			// Create new empty tree for this ancestor.
			current = &object.Tree{}

		case existing.Mode == filemode.Dir:
			// Existing entry is a tree. use it
			hash := existing.Hash
			curr, err := object.GetTree(r.repository.repo.Storer, hash)
			if err != nil {
				return fmt.Errorf("cannot read existing tree %s; root %q, path %q", hash, rootTree.Hash, path)
			}
			current = curr

		default:
			// Existing entry is not a tree. Error.
			return fmt.Errorf("path %q is %s, not a directory in tree %s, root %q", path, existing.Mode, existing.Hash, rootTree.Hash)
		}

		fmt.Println("initializeTrees: setOrAddTreeEntry", name)

		// Set tree in the parent
		setOrAddTreeEntry(parent, object.TreeEntry{
			Name: name,
			Mode: filemode.Dir,
			Hash: plumbing.ZeroHash,
		})

		r.trees[strings.Join(parts[0:i+1], "/")] = current
		parent = current
	}
	// Initialize the package tree.
	lastPart := parts[len(parts)-1]
	fmt.Println("initializeTrees: lastPart", lastPart)
	if !packageTreeHash.IsZero() {
		// Initialize with the supplied package tree.
		packageTree, err := object.GetTree(r.repository.repo.Storer, packageTreeHash)
		if err != nil {
			return fmt.Errorf("cannot find existing package tree %s for package %q: %w", packageTreeHash, packagePath, err)
		}
		r.trees[packagePath] = packageTree
		fmt.Println("initializeTrees: lastPart add entry", lastPart)
		setOrAddTreeEntry(parent, object.TreeEntry{
			Name: lastPart,
			Mode: filemode.Dir,
			Hash: plumbing.ZeroHash,
		})
	} else {
		fmt.Println("initializeTrees: lastPart remove entry")
		// Remove the entry if one exists
		removeTreeEntry(parent, lastPart)
	}

	return nil
}

// storeFile writes a blob with contents at the specified path
func (r *commitHelper) storeFile(path, contents string) error {
	hash, err := r.storeBlob(contents)
	if err != nil {
		return err
	}

	fmt.Println("commit storeFile", path, contents, hash)

	if err := r.storeBlobHashInTrees(path, hash); err != nil {
		return err
	}
	return nil
}

func (r *commitHelper) storeBlob(val string) (plumbing.Hash, error) {
	data := []byte(val)
	eo := r.repository.repo.Storer.NewEncodedObject()
	eo.SetType(plumbing.BlobObject)
	eo.SetSize(int64(len(data)))

	w, err := eo.Writer()
	if err != nil {
		return plumbing.Hash{}, err
	}

	if _, err := w.Write(data); err != nil {
		w.Close()
		return plumbing.Hash{}, err
	}

	if err := w.Close(); err != nil {
		return plumbing.Hash{}, err
	}

	return r.repository.repo.Storer.SetEncodedObject(eo)
}

// storeBlobHashInTrees writes the (previously stored) blob hash at fullpath, marking all the directory trees as dirty.
func (r *commitHelper) storeBlobHashInTrees(fullPath string, hash plumbing.Hash) error {
	dir, file := split(fullPath)
	if file == "" {
		return fmt.Errorf("invalid resource path: %q; no file name", fullPath)
	}

	tree := r.ensureTree(dir)
	setOrAddTreeEntry(tree, object.TreeEntry{
		Name: file,
		Mode: filemode.Regular,
		Hash: hash,
	})

	return nil
}

// ensureTrees ensures we have a trees for all directories in fullPath.
// fullPath is expected to be a directory path.
func (r *commitHelper) ensureTree(fullPath string) *object.Tree {
	if tree, ok := r.trees[fullPath]; ok {
		return tree
	}

	dir, base := split(fullPath)
	parent := r.ensureTree(dir)

	te := object.TreeEntry{
		Name: base,
		Mode: filemode.Dir,
	}

	for ei, ev := range parent.Entries {
		// Replace whole subtrees modified by the package contents.
		if ev.Name == te.Name && !ev.Hash.IsZero() {
			parent.Entries[ei] = te
			goto added
		}
	}
	// Append a new entry
	parent.Entries = append(parent.Entries, te)

added:
	tree := &object.Tree{}
	r.trees[fullPath] = tree
	return tree
}

// storeTrees writes the tree at treePath to git, first writing all child trees.
func (r *commitHelper) storeTrees(treePath string) (plumbing.Hash, error) {
	fmt.Println(r.trees)
	tree, ok := r.trees[treePath]
	if !ok {
		return plumbing.Hash{}, fmt.Errorf("failed to find a tree %q", treePath)
	}

	entries := tree.Entries
	sort.Slice(entries, func(i, j int) bool {
		return entrySortKey(&entries[i]) < entrySortKey(&entries[j])
	})

	// Store all child trees and get their hashes
	for i := range entries {
		e := &entries[i]
		if e.Mode != filemode.Dir {
			continue
		}
		if !e.Hash.IsZero() {
			continue
		}

		hash, err := r.storeTrees(path.Join(treePath, e.Name))
		if err != nil {
			return plumbing.Hash{}, err
		}
		e.Hash = hash
	}

	treeHash, err := r.storeTree(tree)
	if err != nil {
		return plumbing.Hash{}, err
	}

	tree.Hash = treeHash
	return treeHash, nil
}

func (r *commitHelper) storeTree(tree *object.Tree) (plumbing.Hash, error) {
	eo := r.repository.repo.Storer.NewEncodedObject()
	if err := tree.Encode(eo); err != nil {
		return plumbing.Hash{}, err
	}

	treeHash, err := r.repository.repo.Storer.SetEncodedObject(eo)
	if err != nil {
		return plumbing.Hash{}, err
	}
	return treeHash, nil
}

// commit stores all changes in git and creates a commit object.
func (r *commitHelper) commit(ctx context.Context, message string, pkgPath string, additionalParentCommits ...plumbing.Hash) (commit, pkgTree plumbing.Hash, err error) {
	rootTreeHash, err := r.storeTrees("")
	if err != nil {
		fmt.Println("commit: storeTrees failed")
		return plumbing.ZeroHash, plumbing.ZeroHash, err
	}

	/*
		var ui *repository.UserInfo
		if h.userInfoProvider != nil {
			ui = h.userInfoProvider.GetUserInfo(ctx)
		}
	*/

	var parentCommits []plumbing.Hash
	if !r.parentCommitHash.IsZero() {
		parentCommits = append(parentCommits, r.parentCommitHash)
	}
	parentCommits = append(parentCommits, additionalParentCommits...)

	fmt.Println("commit: storeCommit", parentCommits, rootTreeHash, message)
	commitHash, err := r.storeCommit(parentCommits, rootTreeHash, message)
	if err != nil {
		return plumbing.ZeroHash, plumbing.ZeroHash, err
	}
	// Update the parentCommitHash so the correct parent will be used for the
	// next commit.
	r.parentCommitHash = commitHash

	if pkg, ok := r.trees[pkgPath]; ok {
		pkgTree = pkg.Hash
	} else {
		pkgTree = plumbing.ZeroHash
	}

	return commitHash, pkgTree, nil
}

// storeCommit creates and writes a commit object to git.
func (r *commitHelper) storeCommit(parentCommits []plumbing.Hash, tree plumbing.Hash, message string) (plumbing.Hash, error) {
	now := time.Now()
	commit := &object.Commit{
		Author: object.Signature{
			Name:  commitSignatureName,
			Email: commitSignatureEmail,
			When:  now,
		},
		Committer: object.Signature{
			Name:  commitSignatureName,
			Email: commitSignatureEmail,
			When:  now,
		},
		Message:  message,
		TreeHash: tree,
	}

	if len(parentCommits) > 0 {
		commit.ParentHashes = parentCommits
	}

	eo := r.repository.repo.Storer.NewEncodedObject()
	if err := commit.Encode(eo); err != nil {
		return plumbing.Hash{}, err
	}
	hash, err := r.repository.repo.Storer.SetEncodedObject(eo)
	if err != nil {
		return plumbing.Hash{}, err
	}
	fmt.Println("storeCommit", hash)
	return hash, nil
}

// Returns a pointer to the entry if found (by name); nil if not found
func findTreeEntry(tree *object.Tree, name string) *object.TreeEntry {
	for i := range tree.Entries {
		e := &tree.Entries[i]
		if e.Name == name {
			return e
		}
	}
	return nil
}

// setOrAddTreeEntry will overwrite the existing entry (by name) or insert if not present.
func setOrAddTreeEntry(tree *object.Tree, entry object.TreeEntry) {
	for i := range tree.Entries {
		e := &tree.Entries[i]
		if e.Name == entry.Name {
			*e = entry // Overwrite the tree entry
			return
		}
	}
	// Not found. append new
	tree.Entries = append(tree.Entries, entry)
}

// removeTreeEntry will remove the specified entry (by name)
func removeTreeEntry(tree *object.Tree, name string) {
	entries := tree.Entries
	for i := range entries {
		e := &entries[i]
		if e.Name == name {
			tree.Entries = append(entries[:i], entries[i+1:]...)
			return
		}
	}
}

// Git sorts tree entries as though directories have '/' appended to them.
func entrySortKey(e *object.TreeEntry) string {
	if e.Mode == filemode.Dir {
		return e.Name + "/"
	}
	return e.Name
}

// split returns the full directory path and file name
// If there is no directory, it returns an empty directory path and the path as the filename.
func split(path string) (string, string) {
	i := strings.LastIndex(path, "/")
	if i >= 0 {
		return path[:i], path[i+1:]
	}
	return "", path
}
