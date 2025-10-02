package badgerstore

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	badger "github.com/dgraph-io/badger/v4"

	"github.com/guyfedwards/nom/v2/internal/constants"
	"github.com/guyfedwards/nom/v2/internal/store"
)

const (
	itemPrefix          = "item:"
	indexPrefix         = "idx:"
	seqKey              = "seq"
	seqBandwidth        = 100
	DefaultDatabaseName = "nom.badger"
)

type BadgerStore struct {
	path  string
	db    *badger.DB
	batch *badger.Txn
	seq   *badger.Sequence
}

func NewBadgerStore(basePath string, dbName string) (*BadgerStore, error) {
	if dbName == "" {
		dbName = DefaultDatabaseName
	}

	// Add .badger extension if missing
	if filepath.Ext(dbName) == "" {
		dbName = dbName + ".badger"
	}

	err := os.MkdirAll(basePath, 0700)
	if err != nil {
		return nil, fmt.Errorf("NewSQLiteStore: %w", err)
	}
	dbpath := filepath.Join(basePath, dbName)

	// Ensure directory exists
	if err := os.MkdirAll(dbpath, 0755); err != nil {
		return nil, fmt.Errorf("NewBadgerStore: failed to create directory: %w", err)
	}

	opts := badger.DefaultOptions(dbpath)
	opts.Logger = nil // Disable badger's default logger

	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("NewBadgerStore: %w", err)
	}

	// Create sequence for auto-incrementing IDs
	seq, err := db.GetSequence([]byte(seqKey), seqBandwidth)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("NewBadgerStore: failed to get sequence: %w", err)
	}

	return &BadgerStore{
		path: dbpath,
		db:   db,
		seq:  seq,
	}, nil
}

// Close closes the BadgerDB database and releases the sequence
func (bs *BadgerStore) Close() error {
	if bs.seq != nil {
		if err := bs.seq.Release(); err != nil {
			return err
		}
	}
	if bs.db != nil {
		return bs.db.Close()
	}
	return nil
}

// BeginBatch starts a writable transaction for batch operations
func (bs *BadgerStore) BeginBatch() error {
	bs.batch = bs.db.NewTransaction(true)
	return nil
}

// EndBatch commits the batch transaction
func (bs *BadgerStore) EndBatch() error {
	if bs.batch == nil {
		return nil
	}
	err := bs.batch.Commit()
	bs.batch = nil
	if err != nil {
		return fmt.Errorf("EndBatch: %w", err)
	}
	return nil
}

// UpsertItem inserts or updates an item
func (bs *BadgerStore) UpsertItem(item store.Item) error {
	if bs.batch != nil {
		return bs.upsertItem(bs.batch, item)
	}

	txn := bs.db.NewTransaction(true)
	defer txn.Discard()

	if err := bs.upsertItem(txn, item); err != nil {
		return err
	}

	return txn.Commit()
}

func (bs *BadgerStore) upsertItem(txn *badger.Txn, item store.Item) error {
	// Create index key for lookup by feedurl and title
	indexKey := []byte(indexPrefix + item.FeedURL + ":" + item.Title)

	// Check if item already exists
	existingID := 0
	idItem, err := txn.Get(indexKey)
	if err == nil {
		err = idItem.Value(func(val []byte) error {
			existingID = btoi(val)
			return nil
		})
		if err != nil {
			return fmt.Errorf("read existing ID: %w", err)
		}
	} else if err != badger.ErrKeyNotFound {
		return fmt.Errorf("lookup existing item: %w", err)
	}

	if existingID == 0 {
		// Insert new item
		nextID, err := bs.seq.Next()
		if err != nil {
			return fmt.Errorf("get next ID: %w", err)
		}

		// Add 1 to match BoltStore behavior (IDs start at 1, not 0)
		item.ID = int(nextID) + 1
		item.CreatedAt = time.Now()
		item.UpdatedAt = time.Now()

		data, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("marshal item: %w", err)
		}

		itemKey := []byte(itemPrefix + string(itob(item.ID)))
		if err := txn.Set(itemKey, data); err != nil {
			return fmt.Errorf("put item: %w", err)
		}

		// Create index entry
		if err := txn.Set(indexKey, itob(item.ID)); err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	} else {
		// Update existing item
		itemKey := []byte(itemPrefix + string(itob(existingID)))
		existingItem, err := txn.Get(itemKey)
		if err != nil {
			return fmt.Errorf("get existing item: %w", err)
		}

		var existing store.Item
		err = existingItem.Value(func(val []byte) error {
			return json.Unmarshal(val, &existing)
		})
		if err != nil {
			return fmt.Errorf("unmarshal existing item: %w", err)
		}

		existing.Content = item.Content
		existing.UpdatedAt = time.Now()

		data, err := json.Marshal(existing)
		if err != nil {
			return fmt.Errorf("marshal updated item: %w", err)
		}

		if err := txn.Set(itemKey, data); err != nil {
			return fmt.Errorf("update item: %w", err)
		}
	}

	return nil
}

// GetAllItems retrieves all items with the specified ordering
func (bs *BadgerStore) GetAllItems(ordering string) ([]store.Item, error) {
	var items []store.Item

	err := bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(itemPrefix)

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var i store.Item
				if err := json.Unmarshal(val, &i); err != nil {
					return err
				}
				items = append(items, i)
				return nil
			})
			if err != nil {
				continue
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("GetAllItems: %w", err)
	}

	// Sort items by publishedAt or createdAt
	sort.Slice(items, func(i, j int) bool {
		iTime := items[i].PublishedAt
		if iTime.IsZero() {
			iTime = items[i].CreatedAt
		}
		jTime := items[j].PublishedAt
		if jTime.IsZero() {
			jTime = items[j].CreatedAt
		}

		if ordering == constants.DescendingOrdering {
			return iTime.After(jTime)
		}
		return iTime.Before(jTime)
	})

	return items, nil
}

// GetItemByID retrieves an item by its ID
func (bs *BadgerStore) GetItemByID(ID int) (store.Item, error) {
	var item store.Item

	err := bs.db.View(func(txn *badger.Txn) error {
		itemKey := []byte(itemPrefix + string(itob(ID)))
		i, err := txn.Get(itemKey)
		if err != nil {
			return err
		}

		return i.Value(func(val []byte) error {
			return json.Unmarshal(val, &item)
		})
	})
	if err == badger.ErrKeyNotFound {
		return store.Item{}, fmt.Errorf("item not found")
	}
	if err != nil {
		return store.Item{}, fmt.Errorf("GetItemByID: %w", err)
	}

	return item, nil
}

// GetAllFeedURLs retrieves all unique feed URLs
func (bs *BadgerStore) GetAllFeedURLs() ([]string, error) {
	urlMap := make(map[string]bool)

	err := bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(itemPrefix)

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var i store.Item
				if err := json.Unmarshal(val, &i); err != nil {
					return err
				}
				urlMap[i.FeedURL] = true
				return nil
			})
			if err != nil {
				continue
			}
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("GetAllFeedURLs: %w", err)
	}

	urls := make([]string, 0, len(urlMap))
	for url := range urlMap {
		urls = append(urls, url)
	}

	return urls, nil
}

// updateItemField is a helper that handles the common pattern of getting an item,
// modifying it, and saving it back
func (bs *BadgerStore) updateItemField(ID int, updateFn func(*store.Item)) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		itemKey := []byte(itemPrefix + string(itob(ID)))
		i, err := txn.Get(itemKey)
		if err == badger.ErrKeyNotFound {
			return fmt.Errorf("item not found")
		}
		if err != nil {
			return err
		}

		var item store.Item
		err = i.Value(func(val []byte) error {
			return json.Unmarshal(val, &item)
		})
		if err != nil {
			return fmt.Errorf("unmarshal item: %w", err)
		}

		updateFn(&item)

		data, err := json.Marshal(item)
		if err != nil {
			return fmt.Errorf("marshal item: %w", err)
		}

		return txn.Set(itemKey, data)
	})
}

// ToggleRead toggles the read status of an item
func (bs *BadgerStore) ToggleRead(ID int) error {
	return bs.updateItemField(ID, func(item *store.Item) {
		if item.ReadAt.IsZero() {
			item.ReadAt = time.Now()
		} else {
			item.ReadAt = time.Time{}
		}
	})
}

// MarkRead marks an item as read
func (bs *BadgerStore) MarkRead(ID int) error {
	return bs.updateItemField(ID, func(item *store.Item) {
		item.ReadAt = time.Now()
	})
}

// MarkUnread marks an item as unread
func (bs *BadgerStore) MarkUnread(ID int) error {
	return bs.updateItemField(ID, func(item *store.Item) {
		item.ReadAt = time.Time{}
	})
}

// MarkAllRead marks all unread items as read
func (bs *BadgerStore) MarkAllRead() error {
	return bs.db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(itemPrefix)

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.Key()

			err := item.Value(func(val []byte) error {
				var i store.Item
				if err := json.Unmarshal(val, &i); err != nil {
					return err
				}

				if i.ReadAt.IsZero() {
					i.ReadAt = time.Now()
					data, err := json.Marshal(i)
					if err != nil {
						return err
					}
					return txn.Set(key, data)
				}
				return nil
			})
			if err != nil {
				continue
			}
		}

		return nil
	})
}

// ToggleFavourite toggles the favourite status of an item
func (bs *BadgerStore) ToggleFavourite(ID int) error {
	return bs.updateItemField(ID, func(item *store.Item) {
		item.Favourite = !item.Favourite
	})
}

// DeleteByFeedURL deletes all items from a specific feed
func (bs *BadgerStore) DeleteByFeedURL(feedurl string, incFavourites bool) error {
	return bs.db.Update(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(itemPrefix)

		it := txn.NewIterator(opts)
		defer it.Close()

		var toDelete []string
		var indexKeysToDelete []string

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := string(item.Key())

			err := item.Value(func(val []byte) error {
				var i store.Item
				if err := json.Unmarshal(val, &i); err != nil {
					return err
				}

				if i.FeedURL == feedurl {
					if incFavourites || !i.Favourite {
						toDelete = append(toDelete, key)
						// Also track index key for deletion
						indexKey := indexPrefix + i.FeedURL + ":" + i.Title
						indexKeysToDelete = append(indexKeysToDelete, indexKey)
					}
				}
				return nil
			})
			if err != nil {
				continue
			}
		}

		// Delete items and their indices
		for _, key := range toDelete {
			if err := txn.Delete([]byte(key)); err != nil {
				return fmt.Errorf("delete item: %w", err)
			}
		}

		for _, key := range indexKeysToDelete {
			if err := txn.Delete([]byte(key)); err != nil {
				// Index deletion is non-critical, continue
				continue
			}
		}

		return nil
	})
}

// CountUnread counts the number of unread items
func (bs *BadgerStore) CountUnread() (int, error) {
	count := 0

	err := bs.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(itemPrefix)

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			err := item.Value(func(val []byte) error {
				var i store.Item
				if err := json.Unmarshal(val, &i); err != nil {
					return err
				}

				if i.ReadAt.IsZero() {
					count++
				}
				return nil
			})
			if err != nil {
				continue
			}
		}

		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("CountUnread: %w", err)
	}

	return count, nil
}

// Helper functions for integer to byte conversion
func itob(v int) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(v))
	return b
}

func btoi(b []byte) int {
	return int(binary.BigEndian.Uint64(b))
}
