package git

import (
	"fmt"

	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
)

const (
	DefaultMainReferenceName plumbing.ReferenceName = "refs/heads/main"
	OriginName               string                 = "origin"

	MainBranch RefName = "main"

	branchPrefixInLocalRepo  = "refs/remotes/" + OriginName + "/"
	branchPrefixInRemoteRepo = "refs/heads/"
	tagsPrefixInLocalRepo    = "refs/tags/"
	tagsPrefixInRemoteRepo   = "refs/tags/"

	branchRefSpec config.RefSpec = config.RefSpec("+" + branchPrefixInRemoteRepo + "*:" + branchPrefixInLocalRepo + "*")
	tagRefSpec    config.RefSpec = config.RefSpec("+" + tagsPrefixInRemoteRepo + "*:" + tagsPrefixInLocalRepo + "*")
)

var (
	defaultFetchSpec []config.RefSpec = []config.RefSpec{
		branchRefSpec,
		tagRefSpec,
	}
	// DO NOT USE for fetches. Used for reverse reference mapping only.
	reverseFetchSpec []config.RefSpec = []config.RefSpec{
		config.RefSpec(branchPrefixInLocalRepo + "*:" + branchPrefixInRemoteRepo + "*"),
		config.RefSpec(tagsPrefixInLocalRepo + "*:" + tagsPrefixInRemoteRepo + "*"),
	}
)

// RefName represents a relative reference name (i.e. 'main', 'drafts/bucket/v1')
// and supports transformation to the ReferenceName in local (cached) repository
// (those references are in the form 'refs/remotes/origin/...') or in the remote
// repository (those references are in the form 'refs/heads/...').
type RefName string

func (b RefName) RefInRemote() plumbing.ReferenceName {
	return plumbing.ReferenceName(branchPrefixInRemoteRepo + string(b))
}

func (b RefName) RefInLocal() plumbing.ReferenceName {
	return plumbing.ReferenceName(branchPrefixInLocalRepo + string(b))
}

func (b RefName) TagInLocal() plumbing.ReferenceName {
	return plumbing.ReferenceName(tagsPrefixInLocalRepo + string(b))
}

func (b RefName) ForceFetchSpec() config.RefSpec {
	return config.RefSpec(fmt.Sprintf("+%s:%s", b.RefInRemote(), b.RefInLocal()))
}

func refInRemoteFromRefInLocal(n plumbing.ReferenceName) (plumbing.ReferenceName, error) {
	return translateReference(n, reverseFetchSpec)
}

func translateReference(n plumbing.ReferenceName, specs []config.RefSpec) (plumbing.ReferenceName, error) {
	for _, spec := range specs {
		if spec.Match(n) {
			return spec.Dst(n), nil
		}
	}
	return "", fmt.Errorf("cannot translate reference %s", n)
}
