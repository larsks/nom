package badgerstore

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/guyfedwards/nom/v2/internal/constants"
	"github.com/guyfedwards/nom/v2/internal/store"
)

func setupBadgerStore(t *testing.T) (*BadgerStore, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "badgerstore-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	bs, err := NewBadgerStore(tmpDir, "test.db")
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create BadgerStore: %v", err)
	}

	cleanup := func() {
		bs.Close()
		os.RemoveAll(tmpDir)
	}

	return bs, cleanup
}

func TestNewBadgerStore(t *testing.T) {
	bs, cleanup := setupBadgerStore(t)
	defer cleanup()

	if bs == nil {
		t.Fatal("expected store to be created")
	}

	if bs.db == nil {
		t.Fatal("expected db to be initialized")
	}

	if bs.seq == nil {
		t.Fatal("expected sequence to be initialized")
	}
}

func TestBadgerStore_UpsertItem(t *testing.T) {
	bs, cleanup := setupBadgerStore(t)
	defer cleanup()

	item := store.Item{
		Author:      "Test Author",
		Title:       "Test Title",
		FeedURL:     "https://example.com/feed",
		FeedName:    "Example Feed",
		Link:        "https://example.com/article",
		Content:     "Test content",
		PublishedAt: time.Now(),
	}

	err := bs.UpsertItem(item)
	if err != nil {
		t.Fatalf("failed to insert item: %v", err)
	}

	// Verify the item was inserted
	items, err := bs.GetAllItems(constants.DefaultOrdering)
	if err != nil {
		t.Fatalf("failed to get items: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	if items[0].Title != item.Title {
		t.Errorf("expected title %q, got %q", item.Title, items[0].Title)
	}

	if items[0].ID == 0 {
		t.Error("expected ID to be set")
	}

	// Test update
	updatedContent := "Updated content"
	item.Content = updatedContent
	err = bs.UpsertItem(item)
	if err != nil {
		t.Fatalf("failed to update item: %v", err)
	}

	items, err = bs.GetAllItems(constants.DefaultOrdering)
	if err != nil {
		t.Fatalf("failed to get items: %v", err)
	}

	if len(items) != 1 {
		t.Fatalf("expected 1 item after update, got %d", len(items))
	}

	if items[0].Content != updatedContent {
		t.Errorf("expected content %q, got %q", updatedContent, items[0].Content)
	}
}

func TestBadgerStore_BatchOperations(t *testing.T) {
	bs, cleanup := setupBadgerStore(t)
	defer cleanup()

	err := bs.BeginBatch()
	if err != nil {
		t.Fatalf("failed to begin batch: %v", err)
	}

	// Insert multiple items in a batch
	for i := 0; i < 5; i++ {
		item := store.Item{
			Title:   fmt.Sprintf("Test Title %d", i),
			FeedURL: "https://example.com/feed",
			Content: "Test content",
		}
		err := bs.UpsertItem(item)
		if err != nil {
			t.Fatalf("failed to insert item in batch: %v", err)
		}
	}

	err = bs.EndBatch()
	if err != nil {
		t.Fatalf("failed to end batch: %v", err)
	}

	// Verify all items were inserted
	items, err := bs.GetAllItems(constants.DefaultOrdering)
	if err != nil {
		t.Fatalf("failed to get items: %v", err)
	}

	if len(items) != 5 {
		t.Fatalf("expected 5 items, got %d", len(items))
	}
}

func TestBadgerStore_GetItemByID(t *testing.T) {
	bs, cleanup := setupBadgerStore(t)
	defer cleanup()

	item := store.Item{
		Title:   "Test Title",
		FeedURL: "https://example.com/feed",
		Content: "Test content",
	}

	err := bs.UpsertItem(item)
	if err != nil {
		t.Fatalf("failed to insert item: %v", err)
	}

	items, _ := bs.GetAllItems(constants.DefaultOrdering)
	id := items[0].ID

	retrieved, err := bs.GetItemByID(id)
	if err != nil {
		t.Fatalf("failed to get item by ID: %v", err)
	}

	if retrieved.Title != item.Title {
		t.Errorf("expected title %q, got %q", item.Title, retrieved.Title)
	}

	// Test non-existent ID
	_, err = bs.GetItemByID(9999)
	if err == nil {
		t.Error("expected error for non-existent ID")
	}
}

func TestBadgerStore_GetAllFeedURLs(t *testing.T) {
	bs, cleanup := setupBadgerStore(t)
	defer cleanup()

	feeds := []string{
		"https://example.com/feed1",
		"https://example.com/feed2",
		"https://example.com/feed3",
	}

	for _, feedURL := range feeds {
		item := store.Item{
			Title:   "Test Title",
			FeedURL: feedURL,
			Content: "Test content",
		}
		err := bs.UpsertItem(item)
		if err != nil {
			t.Fatalf("failed to insert item: %v", err)
		}
	}

	urls, err := bs.GetAllFeedURLs()
	if err != nil {
		t.Fatalf("failed to get feed URLs: %v", err)
	}

	if len(urls) != len(feeds) {
		t.Fatalf("expected %d feed URLs, got %d", len(feeds), len(urls))
	}
}

func TestBadgerStore_ToggleRead(t *testing.T) {
	bs, cleanup := setupBadgerStore(t)
	defer cleanup()

	item := store.Item{
		Title:   "Test Title",
		FeedURL: "https://example.com/feed",
		Content: "Test content",
	}

	err := bs.UpsertItem(item)
	if err != nil {
		t.Fatalf("failed to insert item: %v", err)
	}

	items, _ := bs.GetAllItems(constants.DefaultOrdering)
	id := items[0].ID

	// Initially unread
	if items[0].Read() {
		t.Error("expected item to be unread initially")
	}

	// Toggle to read
	err = bs.ToggleRead(id)
	if err != nil {
		t.Fatalf("failed to toggle read: %v", err)
	}

	retrieved, _ := bs.GetItemByID(id)
	if !retrieved.Read() {
		t.Error("expected item to be read after toggle")
	}

	// Toggle back to unread
	err = bs.ToggleRead(id)
	if err != nil {
		t.Fatalf("failed to toggle read: %v", err)
	}

	retrieved, _ = bs.GetItemByID(id)
	if retrieved.Read() {
		t.Error("expected item to be unread after second toggle")
	}
}

func TestBadgerStore_MarkAllRead(t *testing.T) {
	bs, cleanup := setupBadgerStore(t)
	defer cleanup()

	// Insert multiple items
	for i := 0; i < 3; i++ {
		item := store.Item{
			Title:   fmt.Sprintf("Test Title %d", i),
			FeedURL: "https://example.com/feed",
			Content: "Test content",
		}
		err := bs.UpsertItem(item)
		if err != nil {
			t.Fatalf("failed to insert item: %v", err)
		}
	}

	// Verify all are unread
	count, err := bs.CountUnread()
	if err != nil {
		t.Fatalf("failed to count unread: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 unread items, got %d", count)
	}

	// Mark all as read
	err = bs.MarkAllRead()
	if err != nil {
		t.Fatalf("failed to mark all read: %v", err)
	}

	// Verify all are read
	count, err = bs.CountUnread()
	if err != nil {
		t.Fatalf("failed to count unread: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 unread items, got %d", count)
	}
}

func TestBadgerStore_ToggleFavourite(t *testing.T) {
	bs, cleanup := setupBadgerStore(t)
	defer cleanup()

	item := store.Item{
		Title:   "Test Title",
		FeedURL: "https://example.com/feed",
		Content: "Test content",
	}

	err := bs.UpsertItem(item)
	if err != nil {
		t.Fatalf("failed to insert item: %v", err)
	}

	items, _ := bs.GetAllItems(constants.DefaultOrdering)
	id := items[0].ID

	// Initially not favourite
	if items[0].Favourite {
		t.Error("expected item to not be favourite initially")
	}

	// Toggle to favourite
	err = bs.ToggleFavourite(id)
	if err != nil {
		t.Fatalf("failed to toggle favourite: %v", err)
	}

	retrieved, _ := bs.GetItemByID(id)
	if !retrieved.Favourite {
		t.Error("expected item to be favourite after toggle")
	}

	// Toggle back
	err = bs.ToggleFavourite(id)
	if err != nil {
		t.Fatalf("failed to toggle favourite: %v", err)
	}

	retrieved, _ = bs.GetItemByID(id)
	if retrieved.Favourite {
		t.Error("expected item to not be favourite after second toggle")
	}
}

func TestBadgerStore_DeleteByFeedURL(t *testing.T) {
	bs, cleanup := setupBadgerStore(t)
	defer cleanup()

	feedURL := "https://example.com/feed"

	// Insert regular item
	item1 := store.Item{
		Title:   "Regular store.Item",
		FeedURL: feedURL,
		Content: "Test content",
	}
	err := bs.UpsertItem(item1)
	if err != nil {
		t.Fatalf("failed to insert item: %v", err)
	}

	// Insert favourite item
	item2 := store.Item{
		Title:   "Favourite store.Item",
		FeedURL: feedURL,
		Content: "Test content",
	}
	err = bs.UpsertItem(item2)
	if err != nil {
		t.Fatalf("failed to insert item: %v", err)
	}

	items, _ := bs.GetAllItems(constants.DefaultOrdering)
	favID := items[1].ID
	err = bs.ToggleFavourite(favID)
	if err != nil {
		t.Fatalf("failed to toggle favourite: %v", err)
	}

	// Delete non-favourites
	err = bs.DeleteByFeedURL(feedURL, false)
	if err != nil {
		t.Fatalf("failed to delete by feed URL: %v", err)
	}

	items, _ = bs.GetAllItems(constants.DefaultOrdering)
	if len(items) != 1 {
		t.Fatalf("expected 1 item remaining, got %d", len(items))
	}

	if items[0].Title != "Favourite store.Item" {
		t.Errorf("expected favourite item to remain, got %q", items[0].Title)
	}

	// Delete including favourites
	err = bs.DeleteByFeedURL(feedURL, true)
	if err != nil {
		t.Fatalf("failed to delete by feed URL: %v", err)
	}

	items, _ = bs.GetAllItems(constants.DefaultOrdering)
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
}

func TestBadgerStore_CountUnread(t *testing.T) {
	bs, cleanup := setupBadgerStore(t)
	defer cleanup()

	// Insert multiple items
	for i := 0; i < 5; i++ {
		item := store.Item{
			Title:   fmt.Sprintf("Test Title %d", i),
			FeedURL: "https://example.com/feed",
			Content: "Test content",
		}
		err := bs.UpsertItem(item)
		if err != nil {
			t.Fatalf("failed to insert item: %v", err)
		}
	}

	// All should be unread
	count, err := bs.CountUnread()
	if err != nil {
		t.Fatalf("failed to count unread: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected 5 unread items, got %d", count)
	}

	// Mark some as read
	items, _ := bs.GetAllItems(constants.DefaultOrdering)
	for i := 0; i < 3; i++ {
		err := bs.ToggleRead(items[i].ID)
		if err != nil {
			t.Fatalf("failed to toggle read: %v", err)
		}
	}

	count, err = bs.CountUnread()
	if err != nil {
		t.Fatalf("failed to count unread: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 unread items, got %d", count)
	}
}

func TestBadgerStore_GetAllItems_Ordering(t *testing.T) {
	bs, cleanup := setupBadgerStore(t)
	defer cleanup()

	now := time.Now()

	// Insert items with different published dates
	items := []store.Item{
		{
			Title:       "Item 1",
			FeedURL:     "https://example.com/feed",
			Content:     "Content 1",
			PublishedAt: now.Add(-2 * time.Hour),
		},
		{
			Title:       "Item 2",
			FeedURL:     "https://example.com/feed",
			Content:     "Content 2",
			PublishedAt: now.Add(-1 * time.Hour),
		},
		{
			Title:       "Item 3",
			FeedURL:     "https://example.com/feed",
			Content:     "Content 3",
			PublishedAt: now,
		},
	}

	for _, item := range items {
		err := bs.UpsertItem(item)
		if err != nil {
			t.Fatalf("failed to insert item: %v", err)
		}
	}

	// Test ascending order (oldest first)
	retrieved, err := bs.GetAllItems(constants.AscendingOrdering)
	if err != nil {
		t.Fatalf("failed to get items: %v", err)
	}

	if len(retrieved) != 3 {
		t.Fatalf("expected 3 items, got %d", len(retrieved))
	}

	if retrieved[0].Title != "Item 1" {
		t.Errorf("expected first item to be 'Item 1', got %q", retrieved[0].Title)
	}

	if retrieved[2].Title != "Item 3" {
		t.Errorf("expected last item to be 'Item 3', got %q", retrieved[2].Title)
	}

	// Test descending order (newest first)
	retrieved, err = bs.GetAllItems(constants.DescendingOrdering)
	if err != nil {
		t.Fatalf("failed to get items: %v", err)
	}

	if retrieved[0].Title != "Item 3" {
		t.Errorf("expected first item to be 'Item 3', got %q", retrieved[0].Title)
	}

	if retrieved[2].Title != "Item 1" {
		t.Errorf("expected last item to be 'Item 1', got %q", retrieved[2].Title)
	}
}

func TestBadgerStore_Close(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "badgerstore-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	bs, err := NewBadgerStore(tmpDir, "test.db")
	if err != nil {
		t.Fatalf("failed to create BadgerStore: %v", err)
	}

	err = bs.Close()
	if err != nil {
		t.Fatalf("failed to close store: %v", err)
	}

	// Verify the database directory exists
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("expected database directory to exist after close")
	}
}
