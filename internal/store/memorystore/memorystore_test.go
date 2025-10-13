package memorystore

import (
	"testing"
	"time"

	"github.com/guyfedwards/nom/v2/internal/constants"
	"github.com/guyfedwards/nom/v2/internal/store"
)

func TestNewMemoryStore(t *testing.T) {
	ms := NewMemoryStore()
	if ms == nil {
		t.Fatal("NewMemoryStore returned nil")
	}
	if ms.items == nil {
		t.Error("items slice not initialized")
	}
	if ms.guidIndex == nil {
		t.Error("guidIndex map not initialized")
	}
	if ms.nextID != 1 {
		t.Errorf("expected nextID to be 1, got %d", ms.nextID)
	}
}

func TestUpsertItem(t *testing.T) {
	ms := NewMemoryStore()

	// Test inserting a new item
	item1 := store.Item{
		Title:       "Test Item 1",
		GUID:        "guid-1",
		FeedURL:     "http://example.com/feed",
		PublishedAt: time.Now(),
	}

	err := ms.UpsertItem(&item1)
	if err != nil {
		t.Fatalf("UpsertItem failed: %v", err)
	}

	if len(ms.items) != 1 {
		t.Errorf("expected 1 item, got %d", len(ms.items))
	}

	if ms.items[0].ID != 1 {
		t.Errorf("expected ID to be 1, got %d", ms.items[0].ID)
	}

	// Test upserting the same item by GUID
	item1Updated := store.Item{
		Title:       "Test Item 1 Updated",
		GUID:        "guid-1",
		FeedURL:     "http://example.com/feed",
		PublishedAt: time.Now(),
	}

	err = ms.UpsertItem(&item1Updated)
	if err != nil {
		t.Fatalf("UpsertItem failed: %v", err)
	}

	if len(ms.items) != 1 {
		t.Errorf("expected 1 item after upsert, got %d", len(ms.items))
	}

	if ms.items[0].Title != "Test Item 1 Updated" {
		t.Errorf("expected title to be updated, got %s", ms.items[0].Title)
	}

	if ms.items[0].ID != 1 {
		t.Errorf("expected ID to remain 1, got %d", ms.items[0].ID)
	}

	// Test inserting another item
	item2 := store.Item{
		Title:       "Test Item 2",
		GUID:        "guid-2",
		FeedURL:     "http://example.com/feed",
		PublishedAt: time.Now(),
	}

	err = ms.UpsertItem(&item2)
	if err != nil {
		t.Fatalf("UpsertItem failed: %v", err)
	}

	if len(ms.items) != 2 {
		t.Errorf("expected 2 items, got %d", len(ms.items))
	}

	if ms.items[1].ID != 2 {
		t.Errorf("expected second item ID to be 2, got %d", ms.items[1].ID)
	}
}

func TestGetAllItems(t *testing.T) {
	ms := NewMemoryStore()

	now := time.Now()
	items := []store.Item{
		{Title: "Item 1", GUID: "guid-1", PublishedAt: now.Add(-2 * time.Hour)},
		{Title: "Item 2", GUID: "guid-2", PublishedAt: now.Add(-1 * time.Hour)},
		{Title: "Item 3", GUID: "guid-3", PublishedAt: now},
	}

	for i := range items {
		if err := ms.UpsertItem(&items[i]); err != nil {
			t.Fatalf("UpsertItem failed: %v", err)
		}
	}

	// Test descending order (default)
	result, err := ms.GetAllItems(constants.DescendingOrdering)
	if err != nil {
		t.Fatalf("GetAllItems failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("expected 3 items, got %d", len(result))
	}

	if result[0].Title != "Item 3" {
		t.Errorf("expected first item to be 'Item 3', got '%s'", result[0].Title)
	}

	// Test ascending order
	result, err = ms.GetAllItems(constants.AscendingOrdering)
	if err != nil {
		t.Fatalf("GetAllItems failed: %v", err)
	}

	if result[0].Title != "Item 1" {
		t.Errorf("expected first item to be 'Item 1', got '%s'", result[0].Title)
	}
}

func TestGetItemByID(t *testing.T) {
	ms := NewMemoryStore()

	item := store.Item{
		Title:   "Test Item",
		GUID:    "guid-1",
		FeedURL: "http://example.com/feed",
	}

	if err := ms.UpsertItem(&item); err != nil {
		t.Fatalf("UpsertItem failed: %v", err)
	}

	// Test getting existing item
	retrieved, err := ms.GetItemByID(1)
	if err != nil {
		t.Fatalf("GetItemByID failed: %v", err)
	}

	if retrieved.Title != "Test Item" {
		t.Errorf("expected title 'Test Item', got '%s'", retrieved.Title)
	}

	// Test getting non-existent item
	_, err = ms.GetItemByID(999)
	if err == nil {
		t.Error("expected error for non-existent item, got nil")
	}
}

func TestGetAllFeedURLs(t *testing.T) {
	ms := NewMemoryStore()

	items := []store.Item{
		{Title: "Item 1", GUID: "guid-1", FeedURL: "http://example.com/feed1"},
		{Title: "Item 2", GUID: "guid-2", FeedURL: "http://example.com/feed2"},
		{Title: "Item 3", GUID: "guid-3", FeedURL: "http://example.com/feed1"},
	}

	for i := range items {
		if err := ms.UpsertItem(&items[i]); err != nil {
			t.Fatalf("UpsertItem failed: %v", err)
		}
	}

	urls, err := ms.GetAllFeedURLs()
	if err != nil {
		t.Fatalf("GetAllFeedURLs failed: %v", err)
	}

	if len(urls) != 2 {
		t.Errorf("expected 2 unique URLs, got %d", len(urls))
	}
}

func TestToggleRead(t *testing.T) {
	ms := NewMemoryStore()

	item := store.Item{
		Title: "Test Item",
		GUID:  "guid-1",
	}

	if err := ms.UpsertItem(&item); err != nil {
		t.Fatalf("UpsertItem failed: %v", err)
	}

	// Toggle to read
	if err := ms.ToggleRead(1); err != nil {
		t.Fatalf("ToggleRead failed: %v", err)
	}

	if ms.items[0].ReadAt.IsZero() {
		t.Error("expected item to be marked as read")
	}

	// Toggle back to unread
	if err := ms.ToggleRead(1); err != nil {
		t.Fatalf("ToggleRead failed: %v", err)
	}

	if !ms.items[0].ReadAt.IsZero() {
		t.Error("expected item to be marked as unread")
	}
}

func TestMarkRead(t *testing.T) {
	ms := NewMemoryStore()

	item := store.Item{
		Title: "Test Item",
		GUID:  "guid-1",
	}

	if err := ms.UpsertItem(&item); err != nil {
		t.Fatalf("UpsertItem failed: %v", err)
	}

	if err := ms.MarkRead(1); err != nil {
		t.Fatalf("MarkRead failed: %v", err)
	}

	if ms.items[0].ReadAt.IsZero() {
		t.Error("expected item to be marked as read")
	}
}

func TestMarkUnread(t *testing.T) {
	ms := NewMemoryStore()

	item := store.Item{
		Title:  "Test Item",
		GUID:   "guid-1",
		ReadAt: time.Now(),
	}

	if err := ms.UpsertItem(&item); err != nil {
		t.Fatalf("UpsertItem failed: %v", err)
	}

	if err := ms.MarkUnread(1); err != nil {
		t.Fatalf("MarkUnread failed: %v", err)
	}

	if !ms.items[0].ReadAt.IsZero() {
		t.Error("expected item to be marked as unread")
	}
}

func TestMarkAllRead(t *testing.T) {
	ms := NewMemoryStore()

	items := []store.Item{
		{Title: "Item 1", GUID: "guid-1"},
		{Title: "Item 2", GUID: "guid-2"},
		{Title: "Item 3", GUID: "guid-3"},
	}

	for i := range items {
		if err := ms.UpsertItem(&items[i]); err != nil {
			t.Fatalf("UpsertItem failed: %v", err)
		}
	}

	if err := ms.MarkAllRead(); err != nil {
		t.Fatalf("MarkAllRead failed: %v", err)
	}

	for i, item := range ms.items {
		if item.ReadAt.IsZero() {
			t.Errorf("expected item %d to be marked as read", i)
		}
	}
}

func TestToggleFavourite(t *testing.T) {
	ms := NewMemoryStore()

	item := store.Item{
		Title: "Test Item",
		GUID:  "guid-1",
	}

	if err := ms.UpsertItem(&item); err != nil {
		t.Fatalf("UpsertItem failed: %v", err)
	}

	// Toggle to favourite
	if err := ms.ToggleFavourite(1); err != nil {
		t.Fatalf("ToggleFavourite failed: %v", err)
	}

	if !ms.items[0].Favourite {
		t.Error("expected item to be marked as favourite")
	}

	// Toggle back
	if err := ms.ToggleFavourite(1); err != nil {
		t.Fatalf("ToggleFavourite failed: %v", err)
	}

	if ms.items[0].Favourite {
		t.Error("expected item to be unmarked as favourite")
	}
}

func TestDeleteByFeedURL(t *testing.T) {
	ms := NewMemoryStore()

	items := []store.Item{
		{Title: "Item 1", GUID: "guid-1", FeedURL: "http://example.com/feed1"},
		{Title: "Item 2", GUID: "guid-2", FeedURL: "http://example.com/feed2", Favourite: true},
		{Title: "Item 3", GUID: "guid-3", FeedURL: "http://example.com/feed1", Favourite: true},
	}

	for i := range items {
		if err := ms.UpsertItem(&items[i]); err != nil {
			t.Fatalf("UpsertItem failed: %v", err)
		}
	}

	// Delete feed1 items, excluding favourites
	if err := ms.DeleteByFeedURL("http://example.com/feed1", false); err != nil {
		t.Fatalf("DeleteByFeedURL failed: %v", err)
	}

	if len(ms.items) != 2 {
		t.Errorf("expected 2 items remaining, got %d", len(ms.items))
	}

	// Delete feed1 items, including favourites
	if err := ms.DeleteByFeedURL("http://example.com/feed1", true); err != nil {
		t.Fatalf("DeleteByFeedURL failed: %v", err)
	}

	if len(ms.items) != 1 {
		t.Errorf("expected 1 item remaining, got %d", len(ms.items))
	}

	if ms.items[0].Title != "Item 2" {
		t.Errorf("expected 'Item 2' to remain, got '%s'", ms.items[0].Title)
	}
}

func TestCountUnread(t *testing.T) {
	ms := NewMemoryStore()

	items := []store.Item{
		{Title: "Item 1", GUID: "guid-1"},
		{Title: "Item 2", GUID: "guid-2", ReadAt: time.Now()},
		{Title: "Item 3", GUID: "guid-3"},
	}

	for i := range items {
		if err := ms.UpsertItem(&items[i]); err != nil {
			t.Fatalf("UpsertItem failed: %v", err)
		}
	}

	count, err := ms.CountUnread()
	if err != nil {
		t.Fatalf("CountUnread failed: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 unread items, got %d", count)
	}
}
