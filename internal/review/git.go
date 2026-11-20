package review

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"path"
	"regexp"
	"strings"
	"time"
	"unicode"
)

var objectID = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

type GitSource struct {
	Commit string
	Path   string
	Blob   string
}

// Git commands have fixed arguments, no shell, no fetch, no checkout and no
// text-conversion filters. Bound stdout even if the local object is oversized.
func gitRead(repo string, limit int64, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", repo, "--no-replace-objects", "--no-lazy-fetch", "--literal-pathspecs"}, args...)...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err = cmd.Start(); err != nil {
		return nil, fmt.Errorf("start Git: %w", err)
	}
	data, readErr := io.ReadAll(io.LimitReader(stdout, limit+1))
	if readErr != nil || int64(len(data)) > limit {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return nil, fmt.Errorf("Git output failed or exceeded %d bytes", limit)
	}
	if err = cmd.Wait(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("Git read timed out")
		}
		return nil, fmt.Errorf("Git object read failed; use Git 2.45+ and confirm the repository, full commit ID and policy path exist locally")
	}
	return data, nil
}

func validPolicyPath(name string) bool {
	if name == "" || len(name) > 1024 || path.IsAbs(name) || path.Clean(name) != name || name == "." || name == ".." || strings.HasPrefix(name, "../") || strings.ContainsAny(name, "\\:") {
		return false
	}
	for _, r := range name {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

// LoadBase reads policy bytes from an exact local commit. The caller must obtain
// that commit ID from a trusted review context. This proves object selection,
// not author identity, branch protection, signature validity or policy approval.
func LoadBase(repo, commit, policyPath string) (GitSource, []byte, error) {
	var source GitSource
	if !objectID.MatchString(commit) {
		return source, nil, fmt.Errorf("base-ref must be a full lowercase 40- or 64-character commit ID; branch names and revision expressions are not accepted")
	}
	if !validPolicyPath(policyPath) {
		return source, nil, fmt.Errorf("base-policy must be a clean repository-relative path using forward slashes")
	}
	kind, err := gitRead(repo, 32, "cat-file", "-t", commit)
	if err != nil {
		return source, nil, err
	}
	if strings.TrimSpace(string(kind)) != "commit" {
		return source, nil, fmt.Errorf("base-ref must identify a commit object")
	}
	entry, err := gitRead(repo, 4096, "ls-tree", "-z", "--full-tree", commit, "--", policyPath)
	if err != nil {
		return source, nil, err
	}
	parts := bytes.Split(entry, []byte{'\t'})
	if len(parts) != 2 || string(parts[1]) != policyPath+"\x00" {
		return source, nil, fmt.Errorf("base policy is not an exact file at the selected commit")
	}
	fields := strings.Fields(string(parts[0]))
	if len(fields) != 3 || (fields[0] != "100644" && fields[0] != "100755") || fields[1] != "blob" || !objectID.MatchString(fields[2]) {
		return source, nil, fmt.Errorf("base policy must be a regular Git blob, not a symlink, directory or submodule")
	}
	data, err := gitRead(repo, 1<<20, "cat-file", "blob", fields[2])
	if err != nil {
		return source, nil, err
	}
	return GitSource{Commit: commit, Path: policyPath, Blob: fields[2]}, data, nil
}
