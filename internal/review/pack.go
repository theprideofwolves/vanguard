package review

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/theprideofwolves/vanguard/internal/gate"
)

type Artifact struct{ Name, SHA256 string }
type Manifest struct {
	Version    string
	BaseSource GitSource
	Artifacts  []Artifact
	Reports    []Artifact
}

// WritePack creates a new directory exclusively. manifest.json is the completion
// marker and is written last. Interrupted/failed output is left for inspection;
// an existing directory is never reused and reports are not copied into the pack.
func WritePack(dir string, result *Result, base, candidate []byte) error {
	if gate.Hash(base) != result.Impact.BasePolicyHash || gate.Hash(candidate) != result.Impact.CandidatePolicyHash {
		return fmt.Errorf("policy snapshots do not match the comparison")
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	files := []struct {
		name string
		data []byte
	}{
		{"review.json", data}, {"review.md", []byte(Markdown(result))},
		{"base-policy.yaml", base}, {"candidate-policy.yaml", candidate},
	}
	if err := os.Mkdir(dir, 0700); err != nil {
		return fmt.Errorf("create a new output directory (parent must exist): %w", err)
	}
	m := Manifest{Version: "vanguard-review-pack/v1", BaseSource: result.BaseSource}
	for _, file := range files {
		if err := writeNew(filepath.Join(dir, file.name), file.data); err != nil {
			return fmt.Errorf("incomplete pack; manifest absent: %w", err)
		}
		m.Artifacts = append(m.Artifacts, Artifact{file.name, gate.Hash(file.data)})
	}
	for _, report := range result.Impact.Reports {
		m.Reports = append(m.Reports, Artifact{report.Name, report.Base.InputHash})
	}
	data, err = json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	if err := writeNew(filepath.Join(dir, "manifest.tmp"), append(data, '\n')); err != nil {
		return err
	}
	return os.Rename(filepath.Join(dir, "manifest.tmp"), filepath.Join(dir, "manifest.json"))
}

func writeNew(name string, data []byte) error {
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	_, writeErr := f.Write(data)
	if writeErr == nil {
		writeErr = f.Sync()
	}
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}
