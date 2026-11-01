// Package github lists the repositories of a GitHub organization. It is the
// only place that holds the user's token: the token goes out in a request
// header and never into an error, a log or a page.
package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const apiURL = "https://api.github.com"

// Repo is one repository as the API describes it.
type Repo struct {
	FullName      string    `json:"full_name"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	Language      string    `json:"language"`
	DefaultBranch string    `json:"default_branch"`
	CloneURL      string    `json:"clone_url"`
	SSHURL        string    `json:"ssh_url"`
	Archived      bool      `json:"archived"`
	PushedAt      time.Time `json:"pushed_at"`
}

// Client talks to the GitHub REST API. An empty Token asks anonymously, which
// only shows public repositories.
type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

// NewClient returns a Client for api.github.com.
func NewClient(token string) *Client {
	return &Client{BaseURL: apiURL, Token: token, HTTP: &http.Client{Timeout: 30 * time.Second}}
}

// APIError is an answer GitHub gave with a status of 300 or more.
type APIError struct {
	Status  int
	Message string
}

func (e *APIError) Error() string { return fmt.Sprintf("github: %d %s", e.Status, e.Message) }

// Repos lists every repository of org that the token reaches, following the
// pagination. When org is not an organization it lists the repositories of
// the user of that name.
func (c *Client) Repos(ctx context.Context, org string) ([]Repo, error) {
	name := url.PathEscape(org)
	repos, err := getAll[Repo](ctx, c, c.BaseURL+"/orgs/"+name+"/repos?per_page=100&type=all")
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusNotFound {
		return repos, err
	}
	// /users/{login}/repos shows public repositories only, even to the token's
	// own owner, so the owner's own account is listed through /user/repos.
	if c.Token != "" {
		if login, err := c.login(ctx); err == nil && strings.EqualFold(login, org) {
			return getAll[Repo](ctx, c, c.BaseURL+"/user/repos?per_page=100&affiliation=owner")
		}
	}
	return getAll[Repo](ctx, c, c.BaseURL+"/users/"+name+"/repos?per_page=100&type=all")
}

// login returns the login of the token's owner.
func (c *Client) login(ctx context.Context) (string, error) {
	body, _, err := c.get(ctx, c.BaseURL+"/user")
	if err != nil {
		return "", err
	}
	var me struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal(body, &me); err != nil {
		return "", fmt.Errorf("github: reading /user: %w", err)
	}
	return me.Login, nil
}

// Orgs lists the account of the token's owner and then its organizations.
// Without a token it returns nothing.
func (c *Client) Orgs(ctx context.Context) ([]string, error) {
	if c.Token == "" {
		return nil, nil
	}
	login, err := c.login(ctx)
	if err != nil {
		return nil, err
	}
	orgs, err := getAll[struct {
		Login string `json:"login"`
	}](ctx, c, c.BaseURL+"/user/orgs?per_page=100")
	if err != nil {
		return nil, err
	}
	logins := []string{login}
	for _, o := range orgs {
		logins = append(logins, o.Login)
	}
	return logins, nil
}

// getAll reads a paginated list, following Link: rel="next".
func getAll[T any](ctx context.Context, c *Client, first string) ([]T, error) {
	var all []T
	for next := first; next != ""; {
		body, link, err := c.get(ctx, next)
		if err != nil {
			return nil, err
		}
		var page []T
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("github: reading a page of results: %w", err)
		}
		all = append(all, page...)
		next = link
	}
	return all, nil
}

// get fetches one URL on the API host and returns the body and the next page.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, string, error) {
	if err := c.sameHost(rawURL); err != nil {
		return nil, "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "trama")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 64<<20))
	if err != nil {
		return nil, "", err
	}
	if res.StatusCode >= 300 {
		return nil, "", c.apiError(res.StatusCode, body)
	}
	return body, nextLink(res.Header.Get("Link")), nil
}

// sameHost refuses a URL that is not on the API host, so a Link header can
// never send the token somewhere else.
func (c *Client) sameHost(rawURL string) error {
	base, err := url.Parse(c.BaseURL)
	if err != nil {
		return err
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return err
	}
	if u.Scheme != base.Scheme || u.Host != base.Host {
		return fmt.Errorf("github: refusing a link to another host: %s", u.Host)
	}
	return nil
}

// apiError builds the error for a failed answer from GitHub's own message.
func (c *Client) apiError(status int, body []byte) error {
	var v struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &v)
	msg := v.Message
	if msg == "" {
		msg = http.StatusText(status)
	}
	if c.Token != "" {
		msg = strings.ReplaceAll(msg, c.Token, "***")
	}
	return &APIError{Status: status, Message: msg}
}

// nextLink returns the rel="next" URL of a Link header, or "".
func nextLink(header string) string {
	for _, part := range strings.Split(header, ",") {
		if !strings.Contains(part, `rel="next"`) {
			continue
		}
		start, end := strings.Index(part, "<"), strings.Index(part, ">")
		if start >= 0 && end > start {
			return part[start+1 : end]
		}
	}
	return ""
}
