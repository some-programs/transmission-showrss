package showrss

import (
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketAdded = []byte("added")
	// bucketTorrents = []byte("torrents")
	// bucketFeeds    = []byte("feeds")
	allBuckets = [][]byte{
		bucketAdded,
		// bucketTorrents,
		// bucketFeeds,
	}
)

// DB .
type DB struct {
	*bolt.DB
}

func NewDB(filename string) (*DB, error) {
	bdb, err := bolt.Open(filename, 0o600, nil)
	if err != nil {
		return nil, err
	}
	db := &DB{bdb}
	for _, v := range allBuckets {
		err := db.Update(func(tx *bolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists(v)
			if err != nil {
				return fmt.Errorf("could not create bucket '%s': %v", string(v), err)
			}
			return nil
		})
		if err != nil {
			db.Close()
			return nil, err
		}
	}
	return db, nil
}

type dbEpisode struct {
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
	Episode Episode   `json:"episode"`
}

func (d dbEpisode) Key() []byte {
	return d.Key()
}

func newDBEpisode(e Episode) (dbEpisode, error) {
	return dbEpisode{
		Created: time.Now(),
		Episode: e,
	}, nil
}
