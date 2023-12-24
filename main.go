package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	repov1alpha1 "github.com/henderiw/git-loader/apis/config/v1alpha1"
	invv1alpha1 "github.com/henderiw/git-loader/apis/inv/v1alpha1"
	"github.com/henderiw/git-loader/pkg/git"
	"github.com/henderiw/git-loader/pkg/git/schema"
	"github.com/henderiw/logger/log"
	"github.com/iptecharch/schema-server/config"
	sschema "github.com/iptecharch/schema-server/schema"
	"sigs.k8s.io/yaml"
)

const (
	rootPath    = "./schemas"
	rootGitPath = rootPath + "/git"
)

func main() {
	os.Exit(runMain())
}

// runMain does the initial setup to setup logging
func runMain() int {
	// init logging
	l := log.NewLogger(&log.HandlerOptions{Name: "logger", AddSource: false})
	slog.SetDefault(l)

	// init context
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	ctx = log.IntoContext(ctx, l)
	log := log.FromContext(ctx)

	if err := runCmd(ctx); err != nil {
		log.Error("cannot run command", "error", err)
		cancel()
		return 1
	}
	return 0
}

func runCmd(ctx context.Context) error {
	args := os.Args
	if len(args) < 2 {
		return fmt.Errorf("cannot run command with an input schema")
	}
	fileName := args[1]
	b, err := os.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("cannot read file: %s, err: %s", fileName, err.Error())
	}
	cr := &invv1alpha1.Schema{}
	if err := yaml.Unmarshal(b, cr); err != nil {
		return fmt.Errorf("cannot unmarshal file: %s, err: %s", fileName, err.Error())
	}

	if err := os.MkdirAll(rootGitPath, 0766); err != nil {
		return err
	}

	dirpath := ""
	if len(cr.Spec.Dirs) > 0 {
		dirpath = cr.Spec.Dirs[0].Src
	}

	gitSpec := &repov1alpha1.GitRepository{
		URL:       cr.Spec.RepositoryURL,
		Ref:       cr.Spec.Ref,
		Directory: dirpath,
	}

	gitRepo, err := git.OpenRepository(ctx, rootGitPath, gitSpec)
	if err != nil {
		return err
	}

	if len(cr.Spec.Schema.Models) != 0 {
		schema := schema.Schema{
			RootPath: rootPath,
			CR:       cr,
		}
		providerPath := cr.Spec.GetBasePath(rootPath)
		if _, err := os.Stat(cr.Spec.GetBasePath(rootPath)); err != nil {
			if err := os.MkdirAll(providerPath, 0766); err != nil {
				return err
			}
			if err := gitRepo.List(ctx, gitSpec.Ref, schema.Copy); err != nil {
				return err
			}
		}

		if _, err := sschema.NewSchema(&config.SchemaConfig{
			Name:        cr.Name,
			Vendor:      cr.Spec.Provider,
			Version:     cr.Spec.Version,
			Files:       cr.Spec.GetNewSchemaBase(rootPath).Models,
			Directories: cr.Spec.GetNewSchemaBase(rootPath).Includes,
			Excludes:    cr.Spec.GetNewSchemaBase(rootPath).Excludes,
		}); err != nil {
			return err
		}
	}

	return nil
}

/*
 git.LoadOptions{
		FilterPrefix: cr.Spec.Git.Directory,
		Recurse: true,
	}
*/

// options
// Copy files
// Get package
// Load Files

/*
  dirs:
  - src: YANG
    dst: .
  schema:
    models:
    - nokia-combined
    includes:
    - ietf
    - nokia-sros-yang-extensions.yang
    excludes: []
*/

/*
  dirs:
  - src: srlinux-yang-models
    dst: .
  schema:
    models:
    - srl_nokia/models
    includes:
    - ietf
    - openconfig/extensions
    - openconfig/openconfig-extensions.yang
    excludes:
    - .*tools.*
*/
