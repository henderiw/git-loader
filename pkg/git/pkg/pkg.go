package pkg

/*
// LoadOptions holds the configuration for walking a git tree
type LoadOptions struct {
	// FilterPrefix restricts loading to a particular subdirectory.
	// If the subdirectory does not exist an empty map is returned
	FilterPrefix string

	// Recurse enables recursive traversal of the git tree.
	Recurse bool
}

// fileList holds a list of files in the git repository
type fileList struct {
	// parent is the gitRepository of which this is part
	parent *gitRepository

	// commit is the commit at which we scanned
	commit *object.Commit

	// packages holds the files we found
	files map[string]*fileListEntry
}

// fileListEntry is a single file found in a git repository
type fileListEntry struct {
	// parent is the packageList of which we are part
	parent *fileList

	// path is the relative path to the root
	path string

	// treeHash is the git-hash of the git tree corresponding to Path
	treeHash plumbing.Hash

	content string
}
*/

/*
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

	fileList, err := r.load(ctx, commit, opts)
	if err != nil {
		return err
	}
	for fileName := range fileList.files {
		fmt.Println("fileName", fileName)

	}
	return nil
}

/*
// load finds the files in the git repository, under commit, if it is exists at path.
func (r *gitRepository) load(ctx context.Context, commit *object.Commit, opts LoadOptions) (*fileList, error) {
	t, err := r.loadInTree(ctx, commit, opts)
	if err != nil {
		return nil, err
	}
	return t, nil
}
*/

/*
// discoverPackagesInTree finds the packages in the git repository, under commit.
// If filterPrefix is non-empty, only packages with the specified prefix will be returned.
// It is not an error if filterPrefix matches no packages or even is not a real directory name;
// we will simply return an empty list of packages.
func (r *gitRepository) loadInTree(ctx context.Context, commit *object.Commit, opt LoadOptions) (*fileList, error) {
	log := log.FromContext(ctx)
	t := &fileList{
		parent: r,
		commit: commit,
		files:  make(map[string]*fileListEntry),
	}

	rootTree, err := commit.Tree()
	if err != nil {
		return nil, fmt.Errorf("cannot resolve commit %v to tree (corrupted repository?): %w", commit.Hash, err)
	}

	if opt.FilterPrefix != "" {
		tree, err := rootTree.Tree(opt.FilterPrefix)
		if err != nil {
			if err == object.ErrDirectoryNotFound {
				// We treat the filter prefix as a filter, the path doesn't have to exist
				log.Info("could not find filterPrefixin commit; returning no files", "filterPrefixin", opt.FilterPrefix, "commit", commit.Hash)
				return t, nil
			} else {
				return nil, fmt.Errorf("error getting tree %s: %w", opt.FilterPrefix, err)
			}
		}
		rootTree = tree
	}
	fmt.Println("rootTree", rootTree)
	fmt.Println("FilterPrefix", opt.FilterPrefix)

	if err := t.loadFiles(ctx, rootTree, opt.FilterPrefix, opt.Recurse); err != nil {
		return nil, err
	}

	log.Info("discovered files", "hash", commit.Hash.String(), "prefix", opt.FilterPrefix, "files", t.files)
	return t, nil
}
*/

/*
// loadFiles is the recursive function we use to traverse the tree and find files.
// tree is the git-tree we are search, treePath is the repo-relative-path to tree.
func (t *fileList) loadFiles(ctx context.Context, tree *object.Tree, treePath string, recurse bool) error {
	//log := log.FromContext(ctx)


	for _, e := range tree.Entries {
		fmt.Println("tree entry", e.Name, "treePath", treePath)
		fmt.Println(string(e.Mode.Bytes()))
		p := path.Join(treePath, e.Name)

		t.files[p] = &fileListEntry{
			path:     treePath,
			treeHash: tree.Hash,
			parent:   t,
		}

	}
	if recurse {
		for _, e := range tree.Entries {
			if e.Mode != filemode.Dir {
				continue
			}

			// This is safe because this function is only called holding the mutex in gitRepository
			dirTree, err := t.parent.repo.TreeObject(e.Hash)
			if err != nil {
				return fmt.Errorf("error getting git tree %v: %w", e.Hash, err)
			}

			if err := t.loadFiles(ctx, dirTree, path.Join(treePath, e.Name), recurse); err != nil {
				return err
			}
		}
	}

	return nil
}
*/

/*
// loadFiles is the recursive function we use to traverse the tree and find files.
// tree is the git-tree we are search, treePath is the repo-relative-path to tree.
func (t *fileList) loadFiles(ctx context.Context, tree *object.Tree, treePath string, recurse bool) error {
	//log := log.FromContext(ctx)

	fit := tree.Files()
	defer fit.Close()
	for {
		file, err := fit.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("failed to load package resources: %w", err)
		}

		content, err := file.Contents()
		if err != nil {
			return fmt.Errorf("failed to read package file contents: %q, %w", file.Name, err)
		}

		fmt.Println("path", filepath.Join(treePath, file.Name))

		t.files[filepath.Join(treePath, file.Name)] = &fileListEntry{
			path:     filepath.Join(treePath, file.Name),
			treeHash: tree.Hash,
			parent:   t,
			content:  content,
		}
	}

	return nil
}
*/
