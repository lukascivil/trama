package changes

import "strings"

// RecordURL returns the GitHub page of path on branch for a GitHub origin
// (https or ssh), and "" for any other remote.
func RecordURL(origin, branch, path string) string {
	o := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSpace(origin), "/"), ".git")
	var repo string
	for _, prefix := range []string{"https://github.com/", "git@github.com:", "ssh://git@github.com/"} {
		if strings.HasPrefix(o, prefix) {
			repo = strings.TrimPrefix(o, prefix)
		}
	}
	if repo == "" {
		return ""
	}
	return "https://github.com/" + repo + "/blob/" + branch + "/" + path
}
