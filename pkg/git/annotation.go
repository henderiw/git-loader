package git

import (
	"encoding/json"
	"fmt"
)

// gitAnnotation is the structured data that we store with commits.
// Currently this is stored as a json-encoded blob in the commit message,
type gitAnnotation struct {
	// PackagePath is the path of the package we modified.
	// This is useful for disambiguating which package we are modifying in a tree of packages,
	// without having to check file paths.
	PackagePath string `json:"package,omitempty"`

	// WorkspaceName holds the workspaceName of the package revision the commit
	// belongs to.
	WorkspaceName string `json:"workspaceName,omitempty"`

	// Revision hold the revision of the package revision the commit
	// belongs to.
	Revision string `json:"revision,omitempty"`

	// Task holds the task we performed, if a task caused the commit.
	//Task *v1alpha1.Task `json:"task,omitempty"`
}

// AnnotateCommitMessage adds the gitAnnotation to the commit message.
func AnnotateCommitMessage(message string, annotation *gitAnnotation) (string, error) {
	b, err := json.Marshal(annotation)
	if err != nil {
		return "", fmt.Errorf("error marshaling annotation: %w", err)
	}

	message += "\n\nannotation:" + string(b) + "\n"

	return message, nil
}
