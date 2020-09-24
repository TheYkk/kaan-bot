package github

type Comment struct {
	Org string
	Repo string
	Number int64
	User string
	IsPullRequest bool
	IssueAuthor string
	State string
}
