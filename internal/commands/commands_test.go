package commands

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/guyfedwards/nom/v2/internal/config"
	"github.com/guyfedwards/nom/v2/internal/store"
)

func TestSearch(t *testing.T) {
	// Create an in-memory store for testing
	s, err := store.NewInMemorySQLiteStore()
	if err != nil {
		t.Fatalf("failed to create in-memory store: %v", err)
	}

	// Add some test items to the store
	testItems := []store.Item{
		{
			Title:    "Introduction to Golang",
			FeedName: "Tech Blog",
			FeedURL:  "https://techblog.com/feed",
			Link:     "https://techblog.com/golang-intro",
			Tags:     []string{"programming", "golang"},
		},
		{
			Title:    "Python Tutorial",
			FeedName: "Dev Blog",
			FeedURL:  "https://devblog.com/feed",
			Link:     "https://devblog.com/python-tutorial",
			Tags:     []string{"programming", "python"},
		},
		{
			Title:    "Breaking News",
			FeedName: "Hacker News",
			FeedURL:  "https://news.com/feed",
			Link:     "https://news.com/breaking",
			Tags:     []string{"news"},
		},
	}

	for i := range testItems {
		err := s.UpsertItem(&testItems[i])
		if err != nil {
			t.Fatalf("failed to insert test item: %v", err)
		}
	}

	// Create runtime config
	runtime := &config.Runtime{
		Config: &config.Config{
			Pager:      "false",
			ListFormat: "",
			ShowRead:   true, // Show all items including read ones for testing
			Filtering: config.FilterConfig{
				DefaultIncludeFeedName: false,
			},
			Feeds: []config.Feed{
				{
					URL:  "https://techblog.com/feed",
					Name: "Tech Blog",
					Tags: []string{"programming", "golang"},
				},
				{
					URL:  "https://devblog.com/feed",
					Name: "Dev Blog",
					Tags: []string{"programming", "python"},
				},
				{
					URL:  "https://news.com/feed",
					Name: "Hacker News",
					Tags: []string{"news"},
				},
			},
		},
		ConfigPath: "/tmp/test-config.yml",
		ConfigDir:  "/tmp",
	}

	cmds := New(runtime, s)

	testCases := []struct {
		name          string
		query         string
		expectError   bool
		expectedCount int
	}{
		{
			name:          "simple text search",
			query:         "golang",
			expectError:   false,
			expectedCount: 1,
		},
		{
			name:          "feed name filter",
			query:         "feed:\"Tech Blog\"",
			expectError:   false,
			expectedCount: 1,
		},
		{
			name:          "tag filter",
			query:         "tag:programming",
			expectError:   false,
			expectedCount: 2,
		},
		{
			name:          "combined filter",
			query:         "tag:python tutorial",
			expectError:   false,
			expectedCount: 1,
		},
		{
			name:          "no matches",
			query:         "nonexistent",
			expectError:   false,
			expectedCount: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := cmds.Search(tc.query)

			w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			io.Copy(&buf, r)
			output := buf.String()

			if tc.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if tc.expectedCount == 0 {
				if !strings.Contains(output, "No matches found") {
					t.Errorf("expected 'No matches found' message, got: %s", output)
				}
			} else {
				lines := strings.Split(strings.TrimSpace(output), "\n")
				actualCount := len(lines)
				if actualCount != tc.expectedCount {
					t.Errorf("expected %d results, got %d\nOutput: %s", tc.expectedCount, actualCount, output)
				}
			}
		})
	}
}
