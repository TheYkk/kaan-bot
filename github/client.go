package github

import (
	"context"
	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

func Login(ctx context.Context, accessToken string) *github.Client {
	if len(accessToken) == 0 {
		return github.NewClient(nil)
	}

	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: accessToken},
	)
	tokenClient := oauth2.NewClient(ctx, tokenSource)

	client := github.NewClient(tokenClient)
	return client
}
