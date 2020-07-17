package title

import (
	"context"
	"github.com/google/go-github/github"
	"kaan-bot/types"
	"regexp"
	"strings"
)

var (
	RetitleRegex = regexp.MustCompile(`(?mi)^/retitle\s*(.*)$`)
)

func Handle(gc *github.Client, line string, req types.IssueCommentOuter) error {
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
		//user   = req.Comment.User
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
		_, _, _ = gc.Issues.CreateComment(ctx, org, repo, number,  &github.IssueComment{
			Body: &com,
		})
	}

	//if req.Action == "" {
	//	pr, err := gc.GetPullRequest(org, repo, number)
	//	if err != nil {
	//		return err
	//	}
	//	pr.Title = newTitle
	//	_, err = gc.EditPullRequest(org, repo, number, pr)
	//	return err
	//}
	//issue, err := gc.GetIssue(org, repo, number)
	//if err != nil {
	//	return err
	//}
	//issue.Title = newTitle
	//_, err = gc.EditIssue(org, repo, number, issue)
	//return err
	return nil
}
