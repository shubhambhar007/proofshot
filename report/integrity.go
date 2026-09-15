package report

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ReportDigest(r Report) string {
	r.Integrity = ""
	data, _ := json.Marshal(r)
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func VerifyReport(r Report) bool {
	return r.Integrity != "" && r.Integrity == ReportDigest(r)
}

func writeManifest(directory string, paths []string) (string, error) {
	manifestPath := filepath.Join(directory, "SHA256SUMS")
	var content strings.Builder
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			return "", err
		}
		hash := sha256.New()
		_, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		if copyErr != nil {
			return "", copyErr
		}
		if closeErr != nil {
			return "", closeErr
		}
		fmt.Fprintf(&content, "%x  %s\n", hash.Sum(nil), filepath.Base(path))
	}
	return manifestPath, os.WriteFile(manifestPath, []byte(content.String()), 0o644)
}

// AddBundleArtifact includes a newly generated file in the existing manifest.
func AddBundleArtifact(directory, path string) error {
	if filepath.Dir(path) != directory || filepath.Base(path) != "recheck-handoff.html" {
		return fmt.Errorf("unexpected bundle artifact %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(data)
	manifest, err := os.OpenFile(filepath.Join(directory, "SHA256SUMS"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer manifest.Close()
	_, err = fmt.Fprintf(manifest, "%x  %s\n", sum, filepath.Base(path))
	return err
}

func VerifyBundle(directory string) error {
	file, err := os.Open(filepath.Join(directory, "SHA256SUMS"))
	if err != nil {
		return fmt.Errorf("open manifest: %w", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "  ", 2)
		if len(parts) != 2 || filepath.Base(parts[1]) != parts[1] {
			return fmt.Errorf("invalid manifest entry %q", scanner.Text())
		}
		data, readErr := os.ReadFile(filepath.Join(directory, parts[1]))
		if readErr != nil {
			return fmt.Errorf("verify %s: %w", parts[1], readErr)
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != parts[0] {
			return fmt.Errorf("integrity check failed for %s", parts[1])
		}
	}
	return scanner.Err()
}

func ReadReport(path string) (Report, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Report{}, err
	}
	var r Report
	if err := json.Unmarshal(data, &r); err != nil {
		return Report{}, err
	}
	return r, nil
}
