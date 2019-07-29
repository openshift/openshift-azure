package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

var releasenoterx = regexp.MustCompile(`(?s)` + "```" + `release-notes?\r\n(.*?)(\r\n)?` + "```")

func env(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

type options struct {
	githubToken    string
	githubOrg      string
	githubRepo     string
	outputFile     string
	startSHA       string
	endSHA         string
	releaseVersion string
}

func (o *options) bindFlags() *flag.FlagSet {
	flags := flag.NewFlagSet("release-notes", flag.ContinueOnError)

	flags.StringVar(&o.githubToken, "github-token", env("GITHUB_TOKEN", ""), "GitHub access token (required).")
	flags.StringVar(&o.githubOrg, "github-org", env("GITHUB_ORG", "openshift"), "Name of github organization.")
	flags.StringVar(&o.githubRepo, "github-repo", env("GITHUB_REPO", "openshift-azure"), "Name of github repository.")
	flags.StringVar(&o.outputFile, "output-file", env("OUTPUT_FILE", "CHANGELOG.md"), "The path to the where the release notes will be printed.")
	flags.StringVar(&o.startSHA, "start-sha", env("START_SHA", ""), "The tag or commit hash to start at.")
	flags.StringVar(&o.endSHA, "end-sha", env("END_SHA", "master"), "The tag or commit hash to end at.")
	flags.StringVar(&o.releaseVersion, "release-version", env("RELEASE_VERSION", ""), "Which release version to tag the entries as.")

	return flags
}

func (o *options) validate() error {
	if o.githubToken == "" {
		return fmt.Errorf("gitHub token must be set via -github-token or $GITHUB_TOKEN")
	}
	if o.startSHA == "" {
		return fmt.Errorf("starting commit hash must be set via -start-sha or $START_SHA")
	}
	if o.endSHA == "" {
		return fmt.Errorf("ending commit hash must be set via -end-sha or $END_SHA")
	}
	if o.releaseVersion == "" {
		return fmt.Errorf("release version must be set via -release-version or $RELEASE_VERSION")
	}

	return nil
}

type note struct {
	commit      *github.RepositoryCommit
	pullRequest *github.PullRequest
	message     *string
}

func (n *note) String() string {
	msg := *n.message
	title := strings.ToUpper(string(msg[0])) + msg[1:]
	number := n.pullRequest.GetNumber()
	url := n.pullRequest.GetHTMLURL()
	author := n.pullRequest.GetUser().GetLogin()
	authorUrl := n.pullRequest.GetUser().GetHTMLURL()
	mergedAt := n.pullRequest.GetMergedAt().Format("02/01/2006")

	return fmt.Sprintf("- %s ([#%d](%s), [@%s](%s), %s)\n", title, number, url, author, authorUrl, mergedAt)
}

type scraper struct {
	*github.Client
	log     *logrus.Entry
	options *options
}

func NewScraper(ctx context.Context, log *logrus.Entry, config *options) (*scraper, error) {
	var cli *http.Client
	cli = oauth2.NewClient(ctx, oauth2.StaticTokenSource(&oauth2.Token{AccessToken: config.githubToken}))
	s := &scraper{
		Client:  github.NewClient(cli),
		log:     log,
		options: config,
	}

	return s, nil
}

func (s *scraper) noteFromCommit(ctx context.Context, commit *github.RepositoryCommit) (*note, error) {
	var prNumber int
	if _, err := fmt.Sscanf(commit.Commit.GetMessage(), "Merge pull request #%d", &prNumber); err != nil {
		return nil, err
	}
	pr, _, err := s.Client.PullRequests.Get(ctx, s.options.githubOrg, s.options.githubRepo, prNumber)
	if err != nil {
		return nil, err
	}

	noteBlocks := releasenoterx.FindStringSubmatch(pr.GetBody())
	if noteBlocks != nil && !(noteBlocks[1] == "" || strings.EqualFold(noteBlocks[1], "none")) {
		return &note{
			commit:      commit,
			pullRequest: pr,
			message:     &noteBlocks[1],
		}, nil
	}
	return nil, fmt.Errorf("no release-notes tag found for commit %s", commit.GetSHA())
}

func (s *scraper) FetchReleaseNotes(ctx context.Context) ([]*note, error) {
	var notes []*note

	startCommit, _, err := s.Repositories.GetCommit(ctx, s.options.githubOrg, s.options.githubRepo, s.options.startSHA)
	if err != nil {
		return nil, err
	}
	endCommit, _, err := s.Repositories.GetCommit(ctx, s.options.githubOrg, s.options.githubRepo, s.options.endSHA)
	if err != nil {
		return nil, err
	}
	listOptions := &github.CommitsListOptions{
		Since: startCommit.Commit.GetCommitter().GetDate(),
		Until: endCommit.Commit.GetCommitter().GetDate(),
	}
	listOptions.PerPage = 100

	for {
		commits, resp, err := s.Client.Repositories.ListCommits(ctx, s.options.githubOrg, s.options.githubRepo, listOptions)
		if err != nil {
			return nil, err
		}
		for _, commit := range commits {
			if len(commit.Parents) > 1 {
				note, err := s.noteFromCommit(ctx, commit)
				if err != nil {
					s.log.Debug(err)
					continue
				}
				notes = append(notes, note)
			}
		}
		if resp.NextPage == 0 {
			break
		}
		listOptions.Page = resp.NextPage
	}

	return notes, nil
}

func (s *scraper) WriteToFile(w io.Writer, notes []*note) error {
	var err error
	write := func(s string) {
		if err != nil {
			return
		}
		_, err = w.Write([]byte(s))
	}
	write(fmt.Sprintf("## %s\n\n", s.options.releaseVersion))
	for _, note := range notes {
		write(note.String())
	}
	write("\n\n")
	return nil
}

func run(ctx context.Context, log *logrus.Entry) error {
	config := &options{}
	flags := config.bindFlags()

	log.Info("parsing flags")
	if err := flags.Parse(os.Args[1:]); err != nil {
		return err
	}

	log.Info("validating config")
	if err := config.validate(); err != nil {
		return err
	}

	log.Info("creating release notes scraper")
	scraper, err := NewScraper(ctx, log, config)
	if err != nil {
		return err
	}

	repo := fmt.Sprintf("github.com/%s/%s", scraper.options.githubOrg, scraper.options.githubRepo)
	log.Infof("fetching release notes %s..%s from %s", scraper.options.startSHA, scraper.options.endSHA, repo)
	notes, err := scraper.FetchReleaseNotes(ctx)
	if err != nil {
		return err
	}

	log.Infof("writing release notes to file %s", scraper.options.outputFile)
	var output io.Writer
	if config.outputFile == "" {
		output = os.Stdout
	} else {
		output, err = os.OpenFile(config.outputFile, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer output.(*os.File).Close()
	}
	err = scraper.WriteToFile(output, notes)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	ctx := context.Background()
	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	log := logrus.NewEntry(logrus.StandardLogger())
	if err := run(ctx, log); err != nil {
		log.Fatal(err)
	}
}
