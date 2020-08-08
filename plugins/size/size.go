package size

import (
	"context"
	"fmt"
	"github.com/google/go-github/github"
	log "github.com/sirupsen/logrus"
	webhook "gopkg.in/go-playground/webhooks.v5/github"
	"strings"
)

type Size struct {
	S   int
	M   int
	L   int
	Xl  int
	Xxl int
}

var defaultSizes = Size{
	S:   10,
	M:   30,
	L:   100,
	Xl:  500,
	Xxl: 1000,
}

func Handle(gc *github.Client, pr webhook.PullRequestPayload) error {
	log.Info("Size plugin")
	if !isPRChanged(pr) {
		return nil
	}

	var (
		owner = pr.PullRequest.Base.Repo.Owner.Login
		repo  = pr.PullRequest.Base.Repo.Name
		num   = pr.PullRequest.Number
	)
	ctx := context.Background()

	changes, _, err := gc.PullRequests.ListFiles(ctx, owner, repo, int(num), &github.ListOptions{})
	if err != nil {
		return fmt.Errorf("can not get PR changes for size plugin: %v", err)
	}

	var count int
	for _, change := range changes {
		count += *change.Additions + *change.Deletions
	}

	labels, _, err := gc.Issues.ListLabelsByIssue(ctx, owner, repo, int(num), &github.ListOptions{})
	if err != nil {
		return fmt.Errorf("while retrieving labels, error: %v", err)
	}
	newLabel := bucket(count, defaultSizes).label()
	var hasLabel bool

	for _, label := range labels {
		if *label.Name == newLabel {
			hasLabel = true
			continue
		}

		if strings.HasPrefix(*label.Name, labelPrefix) {
			if _, err := gc.Issues.RemoveLabelForIssue(ctx, owner, repo, int(num), *label.Name); err != nil {
				return fmt.Errorf("error while removing label %q: %v", label.Name, err)
			}
		}
	}

	if hasLabel {
		return nil
	}

	if _, _, err := gc.Issues.AddLabelsToIssue(ctx, owner, repo, int(num), []string{newLabel}); err != nil {
		return fmt.Errorf("error adding label to %s/%s PR #%d: %v", owner, repo, num, err)
	}
	return nil
}

// One of a set of discrete buckets.
type size int

const (
	sizeXS size = iota
	sizeS
	sizeM
	sizeL
	sizeXL
	sizeXXL
)

const (
	labelPrefix = "size/"

	labelXS      = "size/XS"
	labelS       = "size/S"
	labelM       = "size/M"
	labelL       = "size/L"
	labelXL      = "size/XL"
	labelXXL     = "size/XXL"
	labelUnknown = "size/?"
)

func (s size) label() string {
	switch s {
	case sizeXS:
		return labelXS
	case sizeS:
		return labelS
	case sizeM:
		return labelM
	case sizeL:
		return labelL
	case sizeXL:
		return labelXL
	case sizeXXL:
		return labelXXL
	}

	return labelUnknown
}

func bucket(lineCount int, sizes Size) size {
	if lineCount < sizes.S {
		return sizeXS
	} else if lineCount < sizes.M {
		return sizeS
	} else if lineCount < sizes.L {
		return sizeM
	} else if lineCount < sizes.Xl {
		return sizeL
	} else if lineCount < sizes.Xxl {
		return sizeXL
	}

	return sizeXXL
}

// These are the only actions indicating the code diffs may have changed.
func isPRChanged(pe webhook.PullRequestPayload) bool {
	switch pe.Action {
	case "opened":
		return true
	case "reopened":
		return true
	case "synchronize":
		return true
	case "edited":
		return true
	default:
		return false
	}
}
