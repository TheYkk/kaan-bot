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
	// If closed/merged issues and PRs shouldn't be considered,
	// return early if issue state is not open.
	//if !allowClosedIssues && req.IssueState != "open" {
	//	return nil
	//}
	//
	//// Only consider new comments.
	//if req.Action != github.GenericCommentActionCreated {
	//	return nil
	//}

	// Make sure they are requesting a re-title
	if !RetitleRegex.MatchString(line) {
		return nil
	}

	var (
		org    = req.Repository.Owner.Login
		repo   = req.Repository.Name
		number = req.Issue.Number
		user   = req.Comment.User.Login
	)

	//trusted, err := isTrusted(user)
	//if err != nil {
	//	log.WithError(err).Error("Could not check if user was trusted.")
	//	return err
	//}
	//if !trusted {
	//	return gc.CreateComment(org, repo, number, plugins.FormatResponseRaw(req.Body, req.HTMLURL, user, `Re-titling can only be requested by trusted users, like repository collaborators.`))
	//}
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
