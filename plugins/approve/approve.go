package approve

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	"github.com/sirupsen/logrus"
	webhook "gopkg.in/go-playground/webhooks.v5/github"
	"kaan-bot/plugins/labels"
	"regexp"
	"strings"
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

func Handle(gc *github.Client, line string, req webhook.IssueCommentPayload) error {

	labelMatches := LabelRegex.FindAllStringSubmatch(line, -1)
	removeLabelMatches := RemoveLabelRegex.FindAllStringSubmatch(line, -1)
	customLabelMatches := CustomLabelRegex.FindAllStringSubmatch(line, -1)
	customRemoveLabelMatches := CustomRemoveLabelRegex.FindAllStringSubmatch(line, -1)

	ctx := context.Background()

	var (
		org    = req.Repository.Owner.Login
		repo   = req.Repository.Name
		number = req.Issue.Number
		user   = req.Comment.User.Login

		labelsToAdd    []string
		labelsToRemove []string
	)
	isAuthor, _, err := gc.Organizations.IsMember(ctx, org, user)
	IsCollaborator, _, err := gc.Repositories.IsCollaborator(ctx, org, repo, user)
	perm := false
	if isAuthor || IsCollaborator {
		perm = true
	}
	labelsToAdd = append(getLabelsFromREMatches(labelMatches), getLabelsFromGenericMatches(customLabelMatches, perm)...)
	labelsToRemove = append(getLabelsFromREMatches(removeLabelMatches), getLabelsFromGenericMatches(customRemoveLabelMatches, perm)...)

	if err != nil {
		return err
	}

	// * Add labels to label
	if _, _, err := gc.Issues.AddLabelsToIssue(ctx, org, repo, int(number), labelsToAdd); err != nil {
		return fmt.Errorf("GitHub failed to add the following labels: %v", labelsToAdd)
	}

	// * Remove labels from label
	for _, labelToRemove := range labelsToRemove {
		if _, err := gc.Issues.RemoveLabelForIssue(ctx, org, repo, int(number), labelToRemove); err != nil {
			return fmt.Errorf("GitHub failed to add the following label: %s", labelsToAdd)
		}
	}

	return nil
}

func getLabelsFromREMatches(matches [][]string) (labels []string) {
	for _, match := range matches {
		for _, label := range strings.Split(match[0], " ")[1:] {
			label = strings.ToLower(match[1] + "/" + strings.TrimSpace(label))
			labels = append(labels, label)
		}
	}
	return
}
func getLabelsFromGenericMatches(matches [][]string, perm bool) []string {

	var labels []string

	if !perm {
		return labels
	}

	for _, match := range matches {
		parts := strings.Split(strings.TrimSpace(match[0]), " ")
		if ((parts[0] != "/label") && (parts[0] != "/remove-label")) || len(parts) != 2 {
			continue
		}

		labels = append(labels, parts[1])
	}
	return labels
}
