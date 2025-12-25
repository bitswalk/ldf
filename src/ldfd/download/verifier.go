package download

import (
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// Verifier handles verification of remote resources before download
type Verifier struct {
	httpClient *http.Client
}

// VerificationResult contains the result of a verification check
type VerificationResult struct {
	Exists        bool
	ContentLength int64
	LastModified  time.Time
	ETag          string
	Error         error
}

// NewVerifier creates a new verifier with the given HTTP client
func NewVerifier(httpClient *http.Client) *Verifier {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	return &Verifier{
		httpClient: httpClient,
	}
}

// VerifyRelease checks if a release URL exists using HEAD request
func (v *Verifier) VerifyRelease(ctx context.Context, url string) (*VerificationResult, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return &VerificationResult{Exists: false, Error: err}, nil
	}

	// Set common headers
	req.Header.Set("User-Agent", "ldfd/1.0")

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return &VerificationResult{Exists: false, Error: err}, nil
	}
	defer resp.Body.Close()

	result := &VerificationResult{
		Exists: resp.StatusCode == http.StatusOK,
	}

	if result.Exists {
		result.ContentLength = resp.ContentLength
		result.ETag = resp.Header.Get("ETag")

		if lastMod := resp.Header.Get("Last-Modified"); lastMod != "" {
			if t, err := http.ParseTime(lastMod); err == nil {
				result.LastModified = t
			}
		}
	} else if resp.StatusCode != http.StatusNotFound {
		result.Error = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return result, nil
}

// VerifyGitRef checks if a git reference exists in a remote repository
func (v *Verifier) VerifyGitRef(ctx context.Context, repoURL, ref string) (*VerificationResult, error) {
	// Use git ls-remote to check if the ref exists
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--tags", "--heads", repoURL, ref)

	output, err := cmd.Output()
	if err != nil {
		return &VerificationResult{
			Exists: false,
			Error:  fmt.Errorf("git ls-remote failed: %w", err),
		}, nil
	}

	// If output is not empty, the ref exists
	exists := len(strings.TrimSpace(string(output))) > 0

	return &VerificationResult{
		Exists: exists,
	}, nil
}

// VerifyGitTag checks if a specific tag exists in a remote repository
func (v *Verifier) VerifyGitTag(ctx context.Context, repoURL, tag string) (*VerificationResult, error) {
	// Construct the full ref path for a tag
	fullRef := fmt.Sprintf("refs/tags/%s", tag)

	cmd := exec.CommandContext(ctx, "git", "ls-remote", repoURL, fullRef)

	output, err := cmd.Output()
	if err != nil {
		return &VerificationResult{
			Exists: false,
			Error:  fmt.Errorf("git ls-remote failed: %w", err),
		}, nil
	}

	// If output is not empty, the tag exists
	exists := len(strings.TrimSpace(string(output))) > 0

	return &VerificationResult{
		Exists: exists,
	}, nil
}

// Verify performs verification based on the retrieval method
func (v *Verifier) Verify(ctx context.Context, url string, retrievalMethod string, version string) (*VerificationResult, error) {
	switch retrievalMethod {
	case "release":
		return v.VerifyRelease(ctx, url)
	case "git":
		// For git, we verify the tag exists
		tag := "v" + version
		return v.VerifyGitTag(ctx, url, tag)
	default:
		// Default to release verification
		return v.VerifyRelease(ctx, url)
	}
}
