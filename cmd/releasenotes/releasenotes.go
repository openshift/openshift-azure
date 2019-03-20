package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"gopkg.in/libgit2/git2go.v27"
)

var (
	mergerx       = regexp.MustCompile(`^Merge pull request #(\d+)`)
	releasenoterx = regexp.MustCompile(`(?s)` + "```" + `release-notes?\r\n(.*?)(\r\n)?` + "```")

	reponame    = flag.String("reponame", "openshift/openshift-azure", "GitHub repo name, e.g. openshift/openshift-azure")
	repopath    = flag.String("repopath", ".", "path to local checked out git repo")
	commitrange = flag.String("commitrange", "", "commit range, e.g. v2.5..master")
)

type releasenotes struct {
	gh *github.Client
}

type model struct {
	Commit      *git.Commit
	PR          *github.PullRequest
	ReleaseNote *string
}

func newReleaseNotes(ctx context.Context) (*releasenotes, error) {
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

	return &releasenotes{
		gh: github.NewClient(cli),
	}, nil
}

func (rn *releasenotes) mergeCommits(repopath, commitrange string) ([]*git.Commit, error) {
	repo, err := git.OpenRepository(repopath)
	if err != nil {
		return nil, err
	}

	w, err := repo.Walk()
	if err != nil {
		return nil, err
	}

	w.Sorting(git.SortReverse)

	err = w.PushRange(commitrange)
	if err != nil {
		return nil, err
	}

	var mergeCommits []*git.Commit

	err = w.Iterate(func(commit *git.Commit) bool {
		if commit.ParentCount() > 1 {
			mergeCommits = append(mergeCommits, commit)
		}
		return true
	})
	if err != nil {
		return nil, err
	}

	return mergeCommits, nil
}

func (rn *releasenotes) enrichCommits(ctx context.Context, reponame string, mergeCommits []*git.Commit) []model {
	var models []model
	for _, commit := range mergeCommits {
		m := mergerx.FindStringSubmatch(commit.Message())
		if m == nil {
			log.Printf("couldn't find PR number on commit %s", commit.Id().String())
			continue
		}

		number, err := strconv.Atoi(m[1])
		if err != nil {
			log.Print(err)
			continue
		}

		log.Printf("retrieving PR %d", number)
		pr, _, err := rn.gh.PullRequests.Get(ctx, reponame[:strings.IndexByte(reponame, '/')], reponame[strings.IndexByte(reponame, '/')+1:], number)
		if err != nil {
			log.Printf("couldn't retrieve PR https://github.com/%s/pull/%d", reponame, number)
			continue
		}

		var releasenote *string
		m = releasenoterx.FindStringSubmatch(*pr.Body)
		if m != nil {
			releasenote = &m[1]
		} else {
			log.Printf("couldn't find release note on PR https://github.com/%s/pull/%d", reponame, number)
		}

		models = append(models, model{Commit: commit, PR: pr, ReleaseNote: releasenote})
	}

	return models
}

func (rn *releasenotes) printModels(w io.Writer, reponame string, models []model) error {
	for _, model := range models {
		if model.ReleaseNote == nil || strings.EqualFold(*model.ReleaseNote, "none") {
			continue
		}

		_, err := fmt.Fprintf(w, "## %s ([#%d](https://github.com/%s/pull/%[2]d), [@%[4]s](https://github.com/%[4]s), %[5]s)\n\n",
			*model.PR.Title,
			*model.PR.Number,
			reponame,
			*model.PR.User.Login,
			model.PR.MergedAt.Format("02/01/2006"),
		)
		if err != nil {
			return err
		}

		_, err = fmt.Fprintf(w, "%s\n\n\n",
			*model.ReleaseNote,
		)
		if err != nil {
			return err
		}
	}

	return nil
}

func (rn *releasenotes) run(ctx context.Context) error {
	mergeCommits, err := rn.mergeCommits(*repopath, *commitrange)
	if err != nil {
		return err
	}

	models := rn.enrichCommits(ctx, *reponame, mergeCommits)

	return rn.printModels(os.Stdout, *reponame, models)
}

func main() {
	ctx := context.Background()

	flag.Parse()

	rn, err := newReleaseNotes(ctx)
	if err != nil {
		panic(err)
	}
	if err = rn.run(ctx); err != nil {
		panic(err)
	}
}
