package github

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type Approvers struct {
	Approvers []string `yaml:"approvers"`
}

func IsAuthor(repo, owner string) (bool, error) {
	// ? Parse repo owners
	response, err := http.Get(fmt.Sprintf("https://raw.githubusercontent.com/%s/OWNERS", repo))
	if err != nil {
		return false, err
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return false, err
	}
	trimmedBody := strings.TrimSpace(string(body))
	lines := strings.Split(trimmedBody, "\n")

	for _, line := range lines {
		if line == owner {
			return true, nil
		}
	}

	return false, nil

}
