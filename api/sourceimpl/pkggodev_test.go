package sourceimpl

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/morikuni/failure/v2"
)

// mockHTTPClient creates a test client that returns the content of the specified file
func mockHTTPClient(t *testing.T, filename string) *http.Client {
	t.Helper()

	content, err := os.ReadFile(filepath.Join("testdata", filename))
	if err != nil {
		t.Fatalf("Failed to read test data: %v", err)
	}

	return &http.Client{
		Transport: &mockTransport{
			t:       t,
			content: content,
		},
	}
}

type mockTransport struct {
	t       *testing.T
	content []byte
}

func (m *mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Verify go-get parameter
	if req.URL.Query().Get("go-get") != "1" {
		m.t.Errorf("Expected go-get=1 parameter, got %v", req.URL.Query())
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader("Bad Request")),
		}, nil
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"text/html"},
		},
		Body: io.NopCloser(bytes.NewReader(m.content)),
	}, nil
}

func TestDetectGoMetadata(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		pkgPath     string
		wantRepo    string
		wantHome    string
		wantErrCode any
	}{
		{
			name:        "Valid go-import and go-source meta tags",
			filename:    "go_import_valid.html",
			pkgPath:     "golang.org/x/tools",
			wantRepo:    "https://go.googlesource.com/tools",
			wantHome:    "https://github.com/golang/tools/",
			wantErrCode: nil,
		},
		{
			name:        "Invalid go-import meta tag",
			filename:    "go_import_invalid.html",
			pkgPath:     "golang.org/x/tools",
			wantRepo:    "",
			wantHome:    "",
			wantErrCode: ErrInvalidMetaTag,
		},
		{
			name:        "Missing go-import meta tag",
			filename:    "go_import_missing.html",
			pkgPath:     "golang.org/x/tools",
			wantRepo:    "",
			wantHome:    "",
			wantErrCode: ErrInvalidMetaTag,
		},
		{
			name:        "Valid go-import meta tag without go-source",
			filename:    "go_import_only_valid.html",
			pkgPath:     "golang.org/x/tools",
			wantRepo:    "https://go.googlesource.com/tools",
			wantHome:    "",
			wantErrCode: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock client
			client := mockHTTPClient(t, tt.filename)

			// Use original package path
			pkgPath := tt.pkgPath

			// Run test
			repo, home, err := detectGoMetadata(pkgPath, client)

			// Check error
			if tt.wantErrCode != nil {
				if err == nil {
					t.Errorf("Expected error %v, got nil", tt.wantErrCode)
					return
				}
				if !failure.Is(err, tt.wantErrCode) {
					t.Errorf("Expected error %v, got %v", tt.wantErrCode, err)
				}
				return
			}

			// Check metadata
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Check repository URL
			if tt.wantRepo != "" {
				if repo == nil {
					t.Error("Expected repository URL, got nil")
					return
				}
				if repo.String() != tt.wantRepo {
					t.Errorf("Expected repository URL %v, got %v", tt.wantRepo, repo.String())
				}
			} else if repo != nil {
				t.Errorf("Expected nil repository URL, got %v", repo.String())
			}

			// Check homepage URL
			if tt.wantHome != "" {
				if home == nil {
					t.Error("Expected homepage URL, got nil")
					return
				}
				if home.String() != tt.wantHome {
					t.Errorf("Expected homepage URL %v, got %v", tt.wantHome, home.String())
				}
			} else if home != nil {
				t.Errorf("Expected nil homepage URL, got %v", home.String())
			}
		})
	}
}

func TestDetectGoMetadata_NetworkError(t *testing.T) {
	// Create client that simulates network error
	client := &http.Client{
		Transport: &errorTransport{},
	}

	// Test with error-producing client
	pkgPath := "golang.org/x/tools"
	_, _, err := detectGoMetadata(pkgPath, client)

	// Verify error
	if err == nil {
		t.Error("Expected error, got nil")
		return
	}
	if !failure.Is(err, ErrRepositoryNotFound) {
		t.Errorf("Expected error %v, got %v", ErrRepositoryNotFound, err)
	}
}

type errorTransport struct{}

func (e *errorTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, failure.New(ErrRepositoryNotFound, failure.Message("simulated network error"))
}

func TestDetectGoMetadata_InvalidURL(t *testing.T) {
	// Test with invalid URL
	_, _, err := detectGoMetadata("://invalid-url", nil)

	// Verify error
	if err == nil {
		t.Error("Expected error, got nil")
		return
	}
	if !failure.Is(err, ErrRepositoryNotFound) {
		t.Errorf("Expected error %v, got %v", ErrRepositoryNotFound, err)
	}
}
