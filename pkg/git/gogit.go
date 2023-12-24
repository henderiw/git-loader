package git

import (
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

// initEmptyRepository initializes an empty bare repository
func initEmptyRepository(path string) (*git.Repository, error) {
	isBare := true 
	repo, err := git.PlainInit(path, isBare)
	if err != nil {
		return nil, err
	}
	if err := initializeDefaultBranches(repo); err != nil {
		return nil, err
	}
	return repo, nil
}

// initializeDefaultBranches 
func initializeDefaultBranches(repo *git.Repository) error {
	// Adjust default references
	if err := repo.Storer.RemoveReference(plumbing.Master); err != nil {
		return err
	}
	// gogit points HEAD at a wrong branch; point it at main
	main := plumbing.NewSymbolicReference(plumbing.HEAD, DefaultMainReferenceName)
	if err := repo.Storer.SetReference(main); err != nil {
		return err
	}
	return nil
}

func openRepository(path string) (*git.Repository, error) {
	dot := osfs.New(path)
	storage := filesystem.NewStorage(dot, cache.NewObjectLRUDefault())
	return git.Open(storage, dot)
}

func initializeOrigin(repo *git.Repository, address string) error {
	cfg, err := repo.Config()
	if err != nil {
		return err
	}

	cfg.Remotes[OriginName] = &config.RemoteConfig{
		Name:  OriginName,
		URLs:  []string{address},
		Fetch: defaultFetchSpec,
	}

	if err := repo.SetConfig(cfg); err != nil {
		return err
	}

	return nil
}
