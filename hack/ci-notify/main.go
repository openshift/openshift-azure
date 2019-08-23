package main

/*

A list of available environment variables can be found here:

https://github.com/kubernetes/test-infra/blob/a9f967cb1235916fb5a4eca661fa083413211242/prow/jobs.md#job-environment-variables

For periodics:
- JOB_NAME
- JOB_TYPE
- JOB_SPEC
- BUILD_ID
- PROW_JOB_ID

*/

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var (
	jobName   = flag.String("job-name", "", "Name of the CI job  e.g.  periodic-ci-azure-vmimage")
	comment   = flag.String("comment", "", "The comment to add.")
	owner     = flag.String("owner", "openshift", "Name of github organization.")
	repo      = flag.String("repo", "openshift-azure", "Name of github repository")
	creator   = flag.String("creator", "openshift-azure-robot", "Github user who will send the notifications.")
	duration  = flag.Duration("since", 504*time.Hour, "How many hours in the past to be considered during the search since the last issue.")
	succeeded = flag.Bool("success", false, "Did the ci job succeed?")
)

type client struct {
	gh *github.Client
}

func newClient(token string) *client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	return &client{
		gh: github.NewClient(tc),
	}
}

// listIssues lists github issues that are open and were created by creator. By default
// it searches 3 weeks into the past.
func (c *client) listIssues(ctx context.Context, owner, repo string, since time.Duration) ([]*github.Issue, error) {
	opt := &github.IssueListByRepoOptions{
		State:   "open",
		Creator: *creator,
		Since:   time.Now().Add(-since),
	}

	issues, _, err := c.gh.Issues.ListByRepo(ctx, owner, repo, opt)
	if err != nil {
		return nil, err
	}

	return issues, nil
}

func (c *client) getIssueByName(ctx context.Context, jobName, owner, repo string, since time.Duration) (*github.Issue, error) {
	issues, err := c.listIssues(ctx, owner, repo, since)
	if err != nil {
		return nil, err
	}
	for _, issue := range issues {
		if *issue.Title == jobName {
			return issue, nil
		}
	}
	return nil, err
}

// getJobLogsURL returns the url to the logs for a given CI job. These logs live in a GCS bucket.
func getJobLogsURL(jobName string) string {
	if os.Getenv("BUILD_ID") != "" {

		return fmt.Sprintf("%s/%s/%v", "https://prow.svc.ci.openshift.org/view/gcs/origin-ci-test/logs", jobName, os.Getenv("BUILD_ID"))
	}
	return fmt.Sprintf("https://prow.svc.ci.openshift.org/job-history/origin-ci-test/logs/%s", jobName)
}

// createOrUpdateJobIssue creates or updates (with a comment) a github issue for a given failing CI job.
func (c *client) createOrUpdateJobIssue(ctx context.Context, jobName, comment, owner, repo string, since time.Duration) error {
	issue, err := c.getIssueByName(ctx, jobName, owner, repo, since)
	if err != nil {
		return err
	}

	if issue == nil {
		users := []string{"openshift/sig-azure"}
		fmt.Print("No issue found.  Creating one")
		ir := &github.IssueRequest{
			Title:     to.StringPtr(jobName),
			Body:      to.StringPtr(fmt.Sprintf("The %s build has failed.  Please check the following [link](%s)<br/><br/>%s", jobName, getJobLogsURL(jobName), comment)),
			Assignees: to.StringSlicePtr(users),
		}
		_, _, err = c.gh.Issues.Create(ctx, owner, repo, ir)
		if err != nil {
			return err
		}
	} else {
		// Since we found the issue we need to add a comment
		cmt := &github.IssueComment{
			Body: to.StringPtr(fmt.Sprintf("The %s build has failed.  Please check the following [link](%s).<br/><br/>%s", jobName, getJobLogsURL(jobName), comment)),
			User: &github.User{
				Login: to.StringPtr(*creator),
			},
		}
		_, _, err = c.gh.Issues.CreateComment(ctx, owner, repo, *issue.Number, cmt)
		if err != nil {
			return err
		}
	}

	return nil
}

// closeJobIssue closes a github issue for a given CI job which previously failed but now passes
func (c *client) closeJobIssue(ctx context.Context, jobName, comment, owner, repo string, since time.Duration) error {
	issue, err := c.getIssueByName(ctx, jobName, owner, repo, since)
	if err != nil {
		return err
	}

	if issue != nil {
		// comment on the issue that we are closing it
		cmt := &github.IssueComment{
			Body: to.StringPtr(fmt.Sprintf("[Build](%s) passed.  Closing.<br/><br/>%s", getJobLogsURL(jobName), comment)),
			User: &github.User{
				Login: to.StringPtr(*creator),
			},
		}
		_, _, err = c.gh.Issues.CreateComment(ctx, owner, repo, *issue.Number, cmt)
		if err != nil {
			return err
		}
		// close the issue as we passed the latest vmimage build
		req := &github.IssueRequest{
			State: to.StringPtr("closed"),
		}
		_, _, err = c.gh.Issues.Edit(ctx, owner, repo, *issue.Number, req)
		if err != nil {
			return err
		}
	} else {
		fmt.Printf("No issues open for %s.\n", jobName)
	}

	return nil
}

func main() {
	flag.Parse()

	if *jobName == "" {
		panic("Please specify a job name.")
	}
	token := os.Getenv("OPENSHIFT_AZURE_ROBOT_TOKEN")
	notify := newClient(token)
	ctx := context.Background()

	if *succeeded {
		err := notify.closeJobIssue(ctx, *jobName, *comment, *owner, *repo, *duration)
		if err != nil {
			panic(err)
		}
	} else {
		err := notify.createOrUpdateJobIssue(ctx, *jobName, *comment, *owner, *repo, *duration)
		if err != nil {
			panic(err)
		}
	}
}
