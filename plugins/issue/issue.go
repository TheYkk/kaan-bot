package issue

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	"kaan-bot/types"
	"regexp"
	"strings"
)
var (
	LabelRegex             = regexp.MustCompile(`(?m)^/(area|committee|kind|language|priority|sig|triage|wg)\s*(.*?)\s*$`)
	RemoveLabelRegex       = regexp.MustCompile(`(?m)^/remove-(area|committee|kind|language|priority|sig|triage|wg)\s*(.*?)\s*$`)
	CustomLabelRegex       = regexp.MustCompile(`(?m)^/label\s*(.*?)\s*$`)
	CustomRemoveLabelRegex = regexp.MustCompile(`(?m)^/remove-label\s*(.*?)\s*$`)
)
func Handle(gc *github.Client,line string, req types.IssueCommentOuter) error {
	labelMatches := LabelRegex.FindAllStringSubmatch(line, -1)
	removeLabelMatches := RemoveLabelRegex.FindAllStringSubmatch(line, -1)
	customLabelMatches := CustomLabelRegex.FindAllStringSubmatch(line, -1)
	customRemoveLabelMatches := CustomRemoveLabelRegex.FindAllStringSubmatch(line, -1)

	ctx := context.Background()

	org := req.Repository.Owner.Login
	repo := req.Repository.Name
	var (
		labelsToAdd         []string
		labelsToRemove      []string
	)


	labelsToAdd = append(getLabelsFromREMatches(labelMatches), getLabelsFromGenericMatches(customLabelMatches)...)
	labelsToRemove = append(getLabelsFromREMatches(removeLabelMatches), getLabelsFromGenericMatches(customRemoveLabelMatches)...)

	// * Add labels to issue
	if _, _, err := gc.Issues.AddLabelsToIssue(ctx,org,repo,req.Issue.Number,labelsToAdd); err != nil {
		return fmt.Errorf("GitHub failed to add the following labels: %v", labelsToAdd)
	}

	// * Remove labels from issue
	for _, labelToRemove := range labelsToRemove {
		if _, err := gc.Issues.RemoveLabelForIssue(ctx,org,repo,req.Issue.Number,labelToRemove); err != nil {
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
func getLabelsFromGenericMatches(matches [][]string) []string {

	var labels []string
	for _, match := range matches {
		parts := strings.Split(strings.TrimSpace(match[0]), " ")
		if ((parts[0] != "/label") && (parts[0] != "/remove-label")) || len(parts) != 2 {
			continue
		}

		labels = append(labels, parts[1])
	}
	return labels
}
