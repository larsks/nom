package store

import (
	"time"
)

type Item struct {
	ID          int
	Author      string
	Title       string
	Favourite   bool
	FeedURL     string
	FeedName    string // added from config if set
	Link        string
	GUID        string
	Content     string
	ReadAt      time.Time
	PublishedAt time.Time
	UpdatedAt   time.Time
	CreatedAt   time.Time
}

func (i Item) Read() bool {
	return !i.ReadAt.IsZero()
}

type Store interface {
	UpsertItem(item *Item) error
	BeginBatch() error
	EndBatch() error
	GetAllItems(ordering string) ([]Item, error)
	GetItemByID(ID int) (Item, error)
	GetAllFeedURLs() ([]string, error)
	ToggleRead(ID int) error
	MarkAllRead() error
	ToggleFavourite(ID int) error
	DeleteByFeedURL(feedurl string, incFavourites bool) error
	CountUnread() (int, error)
}
