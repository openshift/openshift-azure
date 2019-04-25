package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
	git "gopkg.in/libgit2/git2go.v27"

	"github.com/openshift/openshift-azure/pkg/util/log"
)

var (
	logLevel   = flag.String("loglevel", "Debug", "valid values are Debug, Info, Warning, Error")
	targetRepo = flag.String("targetrepo", "openshift/openshift-azure", "Target GitHub repo name, e.g. openshift/openshift-azure")
	sourceRepo = flag.String("sourcerepo", "openshift/openshift-azure", "Source GitHub repo name, e.g. openshift/openshift-azure")
	repoPath   = flag.String("repopath", ".", "path to local checked out git repo")
)

var (
	sourceBranch = "content.update"
	prTitle      = "Automated Content Update"
	prAuthor     = "openshift-azure-bot"
	prEmail      = "aos-azure@redhat.com"
	baseBranch   = "master"
)

type giter struct {
	log *logrus.Entry
	gh  *github.Client
}

type change struct {
	oldFile string
	newFile string
}

func newGiter(ctx context.Context, log *logrus.Entry) (*giter, error) {
	var cli *http.Client

	if token, found := os.LookupEnv("GITHUB_TOKEN"); found {
		cli = oauth2.NewClient(
			ctx,
			oauth2.StaticTokenSource(
				&oauth2.Token{AccessToken: token},
			),
		)
	} else {
		return nil, fmt.Errorf("env GITHUB_TOKEN needs to be set with a valid Github Personal Access Token")
	}

	return &giter{
		log: log,
		gh:  github.NewClient(cli),
	}, nil
}

// getRef returns the commit branch reference object if it exists or creates it
// from the base branch before returning it.
func (g *giter) getRef(ctx context.Context) (ref *github.Reference, err error) {
	if ref, _, err = g.gh.Git.GetRef(ctx, strings.Split(*sourceRepo, "/")[0], strings.Split(*sourceRepo, "/")[1], "refs/heads/"+sourceBranch); err == nil {
		return ref, nil
	}

	// We consider that an error means the branch has not been found and needs to
	// be created.
	var baseRef *github.Reference
	if baseRef, _, err = g.gh.Git.GetRef(ctx, strings.Split(*sourceRepo, "/")[0], strings.Split(*sourceRepo, "/")[1], "refs/heads/"+baseBranch); err != nil {
		return nil, err
	}
	newRef := &github.Reference{Ref: github.String("refs/heads/" + sourceBranch), Object: &github.GitObject{SHA: baseRef.Object.SHA}}
	ref, _, err = g.gh.Git.CreateRef(ctx, strings.Split(*sourceRepo, "/")[0], strings.Split(*sourceRepo, "/")[1], newRef)
	return ref, err
}

// getFiles opens local git repository and reads all untracked/uncommited
// files with files and return in format oldFile:newFile
func (g *giter) getFiles() ([]change, error) {
	repo, err := git.OpenRepository(*repoPath)
	if err != nil {
		return nil, err
	}
	l, err := repo.StatusList(&git.StatusOptions{
		Flags: git.StatusOptIncludeUntracked,
	})
	if err != nil {
		return nil, err
	}
	count, err := l.EntryCount()
	if err != nil {
		return nil, err
	}
	g.log.Debugf("untracked files count %d", count)

	var list []change
	for i := 0; i < count; i++ {
		status, err := l.ByIndex(i)
		if err != nil {
			return nil, err
		}

		// if directory, ignore
		if string(status.IndexToWorkdir.NewFile.Path[len(status.IndexToWorkdir.NewFile.Path)-1]) == "/" {
			continue
		}

		list = append(list, change{
			newFile: status.IndexToWorkdir.NewFile.Path,
			oldFile: status.IndexToWorkdir.OldFile.Path,
		})
	}
	return list, nil
}

// getTree generates the tree to commit based on the given files and the commit
// of the ref you got in getRef.
func (g *giter) getTree(ctx context.Context, ref *github.Reference) (tree *github.Tree, err error) {
	// Create a tree with what to commit.
	entries := []github.TreeEntry{}
	files, err := g.getFiles()
	if err != nil {
		return nil, err
	}

	// Load each file into the tree.
	for _, file := range files {
		name, content, err := getFileContent(file)
		fmt.Println(file)
		if err != nil {
			return nil, err
		}
		entries = append(entries, github.TreeEntry{
			Path:    github.String(name),
			Type:    github.String("blob"),
			Content: github.String(string(content)),
			Mode:    github.String("100644"),
		})
	}

	tree, _, err = g.gh.Git.CreateTree(ctx, strings.Split(*sourceRepo, "/")[0], strings.Split(*sourceRepo, "/")[1], *ref.Object.SHA, entries)
	return tree, err
}

// getFileContent loads the local content of a file and return the target name
// of the file in the target repository and its contents.
func getFileContent(file change) (targetName string, b []byte, err error) {
	if file.oldFile == "" && file.newFile == "" {
		return "", nil, errors.New("no files to commit")
	}

	if _, err := os.Stat(file.newFile); os.IsNotExist(err) {
		b = []byte{}
	} else {
		b, err = ioutil.ReadFile(file.newFile)
	}
	return file.newFile, b, err
}

// pushCommit creates the commit in the given reference using the given tree.
func (g *giter) pushCommit(ctx context.Context, ref *github.Reference, tree *github.Tree) (err error) {
	// Get the parent commit to attach the commit to.
	parent, _, err := g.gh.Repositories.GetCommit(ctx, strings.Split(*sourceRepo, "/")[0], strings.Split(*sourceRepo, "/")[1], *ref.Object.SHA)
	if err != nil {
		return err
	}
	// This is not always populated, but is needed.
	parent.Commit.SHA = parent.SHA

	// Create the commit using the tree.
	date := time.Now()
	author := &github.CommitAuthor{
		Date:  &date,
		Name:  github.String(prAuthor),
		Email: github.String(prEmail),
	}
	commit := &github.Commit{
		Author:  author,
		Tree:    tree,
		Message: github.String(prTitle),
		Parents: []github.Commit{*parent.Commit},
	}
	newCommit, _, err := g.gh.Git.CreateCommit(ctx, strings.Split(*sourceRepo, "/")[0], strings.Split(*sourceRepo, "/")[1], commit)
	if err != nil {
		return err
	}

	// Attach the commit to the master branch.
	ref.Object.SHA = newCommit.SHA
	_, _, err = g.gh.Git.UpdateRef(ctx, strings.Split(*sourceRepo, "/")[0], strings.Split(*sourceRepo, "/")[1], ref, false)
	return err
}

// createPR creates a pull request. Based on: https://godoc.org/github.com/google/go-github/github#example-PullRequestsService-Create
func (g *giter) createPR(ctx context.Context) (err error) {
	newPR := &github.NewPullRequest{
		Title:               github.String(prTitle),
		Head:                github.String(sourceBranch),
		Base:                github.String(baseBranch),
		MaintainerCanModify: github.Bool(true),
	}

	pr, _, err := g.gh.PullRequests.Create(ctx, strings.Split(*targetRepo, "/")[0], strings.Split(*targetRepo, "/")[1], newPR)
	if err != nil {
		return err
	}

	fmt.Printf("PR created: %s\n", pr.GetHTMLURL())
	return nil
}

func (g *giter) run(ctx context.Context) error {
	ref, err := g.getRef(ctx)
	if err != nil {
		g.log.Fatalf("Unable to get/create the commit reference: %s\n", err)
	}
	if ref == nil {
		g.log.Fatalf("No error where returned but the reference is nil")
	}

	tree, err := g.getTree(ctx, ref)
	if err != nil {
		g.log.Fatalf("Unable to create the tree based on the provided files: %s\n", err)
	}

	if err := g.pushCommit(ctx, ref, tree); err != nil {
		g.log.Fatalf("Unable to create the commit: %s\n", err)
	}

	if err := g.createPR(ctx); err != nil {
		g.log.Debug("Error while creating the pull request: %s", err)
	}

	return nil
}

func main() {
	ctx := context.Background()
	flag.Parse()
	logger := logrus.New()
	logger.Formatter = &logrus.TextFormatter{FullTimestamp: true}
	logger.SetLevel(log.SanitizeLogLevel(*logLevel))
	log := logrus.NewEntry(logger)
	log.Info("giter starting")

	g, err := newGiter(ctx, log)
	if err != nil {
		panic(err)
	}
	if err = g.run(ctx); err != nil {
		panic(err)
	}
}
