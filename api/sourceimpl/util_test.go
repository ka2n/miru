package sourceimpl

import "testing"

func TestCleanupRepositoryURL(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "GitHub HTTPS URL",
			url:  "https://github.com/org/repo.git",
			want: "https://github.com/org/repo",
		},
		{
			name: "GitHub git+https URL",
			url:  "git+https://github.com/org/repo.git",
			want: "https://github.com/org/repo",
		},
		{
			name: "GitHub SSH URL",
			url:  "git@github.com:org/repo.git",
			want: "https://github.com/org/repo",
		},
		{
			name: "GitHub SSH URL with ssh:// protocol",
			url:  "ssh://git@github.com/org/repo.git",
			want: "https://github.com/org/repo",
		},
		{
			name: "GitHub git protocol URL",
			url:  "git://github.com/org/repo.git",
			want: "https://github.com/org/repo",
		},
		{
			name: "GitLab HTTPS URL with /-/",
			url:  "https://gitlab.com/org/repo/-/tree/main",
			want: "https://gitlab.com/org/repo/tree/main",
		},
		{
			name: "GitLab SSH URL",
			url:  "git@gitlab.com:org/repo.git",
			want: "https://gitlab.com/org/repo",
		},
		{
			name: "Other hosting service HTTPS URL",
			url:  "https://gitea.example.com/org/repo.git",
			want: "https://gitea.example.com/org/repo",
		},
		{
			name: "Other hosting service SSH URL",
			url:  "git@gitea.example.com:org/repo.git",
			want: "https://gitea.example.com/org/repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanupRepositoryURL(tt.url); got != tt.want {
				t.Errorf("CleanupRepositoryURL() = %v, want %v", got, tt.want)
			}
		})
	}
}
