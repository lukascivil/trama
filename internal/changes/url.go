package changes

import "strings"

// GitHubSlug returns "owner/repo" for a GitHub origin (https or ssh), and
// "" for any other remote.
func GitHubSlug(origin string) string {
	o := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSpace(origin), "/"), ".git")
	for _, prefix := range []string{"https://github.com/", "git@github.com:", "ssh://git@github.com/"} {
		if strings.HasPrefix(o, prefix) {
			return strings.TrimPrefix(o, prefix)
		}
	}
	return ""
}

// RecordURL returns the GitHub page of path on branch for a GitHub origin
// (https or ssh), and "" for any other remote.
func RecordURL(origin, branch, path string) string {
	repo := GitHubSlug(origin)
	if repo == "" {
		return ""
	}
	return "https://github.com/" + repo + "/blob/" + branch + "/" + path
}
