package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/morikuni/failure/v2"
)

const (
	// ErrNPMREADMENotFound represents an error when README is not found
	ErrNPMREADMENotFound ErrorCode = "NPMREADMENotFound"
)

// npmPackageInfo represents the npm package information from registry
type npmPackageInfo struct {
	Readme string `json:"readme"`
}

// FetchNPMReadme fetches the README content from npm registry
func FetchNPMReadme(pkgPath string) (string, error) {
	// Get package information from npm registry
	url := fmt.Sprintf("https://registry.npmjs.org/%s", pkgPath)
	resp, err := http.Get(url)
	if err != nil {
		return "", failure.Wrap(err)
	}
	defer resp.Body.Close()

	// Parse JSON response
	var info npmPackageInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return "", failure.Wrap(err)
	}

	if info.Readme == "" {
		return "", failure.New(ErrNPMREADMENotFound,
			failure.Message("README not found in package"),
			failure.Context{
				"pkg": pkgPath,
			},
		)
	}

	return info.Readme, nil
}
