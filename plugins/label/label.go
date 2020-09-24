package label

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	log "github.com/sirupsen/logrus"
	ghclient "kaan-bot/github"
	"regexp"
	"strings"
)

var (
	LabelRegex             = regexp.MustCompile(`(?m)^/(area|committee|kind|language|priority|sig|triage|wg)\s*(.*?)\s*$`)
	RemoveLabelRegex       = regexp.MustCompile(`(?m)^/remove-(area|committee|kind|language|priority|sig|triage|wg)\s*(.*?)\s*$`)
	CustomLabelRegex       = regexp.MustCompile(`(?m)^/label\s*(.*?)\s*$`)
	CustomRemoveLabelRegex = regexp.MustCompile(`(?m)^/remove-label\s*(.*?)\s*$`)
)

func Handle(gc *github.Client, line string, comment ghclient.Comment) error {
	log.Info("Label plugin")
	labelMatches := LabelRegex.FindAllStringSubmatch(line, -1)
	removeLabelMatches := RemoveLabelRegex.FindAllStringSubmatch(line, -1)
	customLabelMatches := CustomLabelRegex.FindAllStringSubmatch(line, -1)
	customRemoveLabelMatches := CustomRemoveLabelRegex.FindAllStringSubmatch(line, -1)

	ctx := context.Background()

	var (
		org    = comment.Org
		repo   = comment.Repo
		number = comment.Number
		user   = comment.User

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
