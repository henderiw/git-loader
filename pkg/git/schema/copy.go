package schema

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5/plumbing/object"
	invv1alpha1 "github.com/henderiw/git-loader/apis/inv/v1alpha1"
	"github.com/henderiw/logger/log"
)

type Schema struct {
	RootPath string
	CR       *invv1alpha1.Schema
}

func (r *Schema) Copy(ctx context.Context, tree *object.Tree) error {
	log := log.FromContext(ctx)
	providerVersionBasePath := filepath.Join(r.RootPath, r.CR.Spec.Provider, r.CR.Spec.Version)

	fit := tree.Files()
	defer fit.Close()
	for {
		file, err := fit.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return fmt.Errorf("failed to load package resources: %w", err)
		}
		filePath := filepath.Join(providerVersionBasePath, file.Name)
		fmt.Println("copy file", "from", file.Name, "to", filePath)
		content, err := file.Contents()
		if err != nil {
			log.Info("cannot read file", "fileName", file.Name, "error", err.Error())
			continue // we continue although we cannot read file
		}

		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			log.Info("cannot write file", "fileName", filePath, "error", err.Error())
		}

		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			log.Info("cannot write file", "fileName", filePath, "error", err.Error())
		}

	}
	return nil
}
