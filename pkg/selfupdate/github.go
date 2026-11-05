package selfupdate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var ErrNoReleases = errors.New("selfupdate: repo belum punya rilis resmi di GitHub")
var validTag = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+$`)

type Release struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	HTMLURL     string `json:"html_url"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	Prerelease  bool   `json:"prerelease"`
	Draft       bool   `json:"draft"`
}

func FetchLatestRelease(owner, repo string) (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repo)
	client := &http.Client{Timeout: 8 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "gowms-selfupdate")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gagal menghubungi GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNoReleases
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("GitHub API mengembalikan status %d: %s", resp.StatusCode, string(body))
	}

	var releases []Release
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&releases); err != nil {
		return nil, fmt.Errorf("gagal membaca response GitHub: %w", err)
	}

	var latest *Release
	for _, r := range releases {
		if r.Draft || r.Prerelease || !validTag.MatchString(r.TagName) {
			continue
		}
		if !validReleaseURL(r.HTMLURL, owner, repo) {
			continue
		}
		candidate := r
		if latest == nil || CompareVersions(candidate.TagName, latest.TagName) > 0 {
			latest = &candidate
		}
	}
	if latest == nil {
		return nil, ErrNoReleases
	}
	return latest, nil
}

func validReleaseURL(raw, owner, repo string) bool {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host != "github.com" {
		return false
	}
	return u.Path == fmt.Sprintf("/%s/%s/releases/tag/%s", owner, repo, u.Path[strings.LastIndex(u.Path, "/")+1:])
}

func CompareVersions(a, b string) int {
	segA := normalizeVersionSegments(a)
	segB := normalizeVersionSegments(b)
	maxLen := len(segA)
	if len(segB) > maxLen {
		maxLen = len(segB)
	}
	for i := 0; i < maxLen; i++ {
		va, vb := 0, 0
		if i < len(segA) {
			va = segA[i]
		}
		if i < len(segB) {
			vb = segB[i]
		}
		if va != vb {
			if va < vb {
				return -1
			}
			return 1
		}
	}
	return 0
}

func normalizeVersionSegments(v string) []int {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "v")
	v = strings.TrimPrefix(v, "V")
	parts := strings.Split(v, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		numEnd := 0
		for numEnd < len(p) && p[numEnd] >= '0' && p[numEnd] <= '9' {
			numEnd++
		}
		if numEnd == 0 {
			out = append(out, 0)
			continue
		}
		n, err := strconv.Atoi(p[:numEnd])
		if err != nil {
			n = 0
		}
		out = append(out, n)
	}
	return out
}
