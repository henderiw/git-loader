package git

import (
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
)

type RefName string

func (b RefName) RefInLocal() plumbing.ReferenceName {
	return plumbing.ReferenceName(branchPrefixInLocalRepo + string(b))
}

func (b RefName) TagInLocal() plumbing.ReferenceName {
	return plumbing.ReferenceName(tagsPrefixInLocalRepo + string(b))
}