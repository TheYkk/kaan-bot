package title

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	webhook "gopkg.in/go-playground/webhooks.v5/github"
	"regexp"
	"strings"
)

var (
	RetitleRegex = regexp.MustCompile(`(?mi)^/retitle\s*(.*)$`)
)

func Handle(gc *github.Client, line string, req webhook.IssueCommentPayload) error {
	// ? Make sure they are requesting a re-title
	if !RetitleRegex.MatchString(line) {
		return nil
	}

	var (
		org    = req.Repository.Owner.Login
		repo   = req.Repository.Name
		number = req.Issue.Number
		user   = req.Comment.User.Login
	)
	ctx := context.Background()

	matches := RetitleRegex.FindStringSubmatch(line)

	if matches == nil {
		// this shouldn't happen since we checked above
		return nil
	}
	newTitle := strings.TrimSpace(matches[1])

	if newTitle == "" {
		com := "Titles may not be empty."
		_, _, _ = gc.Issues.CreateComment(ctx, org, repo, int(number), &github.IssueComment{
			Body: &com,
		})
	}
	isAuthor, _, err := gc.Organizations.IsMember(ctx, org, user)
	IsCollaborator, _, err := gc.Repositories.IsCollaborator(ctx, org, repo, user)

	if !isAuthor && !IsCollaborator {
		com := fmt.Sprintf(`Hi @%s Re-titling can only be requested by trusted users, like repository collaborators.`, user)
		_, _, err = gc.Issues.CreateComment(ctx, org, repo, int(number), &github.IssueComment{
			Body: &com,
		})

		if err != nil {
			return err
		}
		return nil
	}

	// ? Is pull request
	if req.Issue.PullRequest.URL != "" {
		_, _, err := gc.PullRequests.Edit(ctx, org, repo, int(number), &github.PullRequest{
			Title: &newTitle,
		})
		if err != nil {
			return err
		}

	} else {

		_, _, err := gc.Issues.Edit(ctx, org, repo, int(number), &github.IssueRequest{
			Title: &newTitle,
		})

		if err != nil {
			return err
		}
	}
	return nil
}
