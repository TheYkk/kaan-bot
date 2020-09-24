package lgtm

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	webhook "gopkg.in/go-playground/webhooks.v5/github"
	ghclient "kaan-bot/github"
	"kaan-bot/plugins/labels"
	"regexp"
)

var (
	addLGTMLabelNotification   = "LGTM label has been added.  <details>Git tree hash: %s</details>"
	addLGTMLabelNotificationRe = regexp.MustCompile(fmt.Sprintf(addLGTMLabelNotification, "(.*)"))
	configInfoReviewActsAsLgtm = `Reviews of "approve" or "request changes" act as adding or removing LGTM.`
	configInfoStoreTreeHash    = `Squashing commits does not remove LGTM.`
	// LGTMLabel is the name of the lgtm label applied by the lgtm plugin
	LGTMLabel = labels.LGTM
	// LGTMRe is the regex that matches lgtm comments
	LGTMRe = regexp.MustCompile(`(?mi)^/lgtm(?: no-issue)?\s*$`)
	// LGTMCancelRe is the regex that matches lgtm cancel comments
	LGTMCancelRe        = regexp.MustCompile(`(?mi)^/lgtm cancel\s*$`)
	removeLGTMLabelNoti = "New changes are detected. LGTM label has been removed."
)

func RemoveLabel(gc *github.Client, req webhook.PullRequestPayload) error {
	if req.Action != "synchronize" {
		return nil
	}
	ctx := context.Background()

	var (
		org    = req.Repository.Owner.Login
		repo   = req.Repository.Name
		number = req.PullRequest.Number
	)

	if _, err := gc.Issues.RemoveLabelForIssue(ctx, org, repo, int(number), LGTMLabel); err != nil {
		return fmt.Errorf("GitHub failed to remove the following label: %s", LGTMLabel)
	}

	logrus.Infof("Commenting with \"%s\".", removeLGTMLabelNoti)

	_, _, err := gc.Issues.CreateComment(ctx, org, repo, int(number), &github.IssueComment{
		Body: &removeLGTMLabelNoti,
	})

	if err != nil {
		return err
	}

	return nil
}

func Handle(gc *github.Client, line string, comment ghclient.Comment) error {
	ctx := context.Background()

	var (
		org    = comment.Org
		repo   = comment.Repo
		number = comment.Number
		user   = comment.User
		issueAuthor = comment.IssueAuthor
	)

	isAuthor, _, err := gc.Organizations.IsMember(ctx, org, user)
	IsCollaborator, _, err := gc.Repositories.IsCollaborator(ctx, org, repo, user)

	if !isAuthor && !IsCollaborator {
		return nil
	}

	// ? If is not pr return
	if !comment.IsPullRequest || comment.State != "open" {
		return nil
	}

	//? Author cannot LGTM own PR, comment and abort
	isOwnPR := user == issueAuthor
	if isOwnPR {
		resp := "@"+ user +" you cannot LGTM your own PR."
		logrus.Infof("Commenting with \"%s\".", resp)
		_, _, err = gc.Issues.CreateComment(ctx, org, repo, int(number), &github.IssueComment{
			Body: &resp,
		})
		return err
	}

	// ? Assign commentor to issue
	logrus.Infof("Assigning %s/%s#%d to %s", org, repo, number, user)
	if _, _, err := gc.Issues.AddAssignees(ctx, org, repo, int(number), []string{user}); err != nil {
		logrus.WithError(err).Errorf("Failed to assign %s/%s#%d to %s", org, repo, number, user)
	}

	// ? If lgtm cancel commented remove label
	if LGTMCancelRe.MatchString(line) {
		if _, err := gc.Issues.RemoveLabelForIssue(ctx, org, repo, int(number), LGTMLabel); err != nil {
			return fmt.Errorf("GitHub failed to remove the following label: %s", LGTMLabel)
		}
	}
	// ? Add lgtm label
	if LGTMRe.MatchString(line) {
		if _, _, err := gc.Issues.AddLabelsToIssue(ctx, org, repo, int(number), []string{LGTMLabel}); err != nil {
			return fmt.Errorf("GitHub failed to add the following label: %v", LGTMLabel)
		}
	}

	return nil
}
