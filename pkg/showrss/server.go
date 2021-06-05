package showrss

import (
	"net/http"

	"github.com/pborzenkov/go-transmission/transmission"
	"github.com/some-programs/transmission-showrss/pkg/log"
	bolt "go.etcd.io/bbolt"
)

func APIServer(db *DB, tc *transmission.Client, bindAddr string) error {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		first := true
		w.Write([]byte("["))
		err := db.View(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(bucketAdded)
			c := bucket.Cursor()
			for k, v := c.First(); k != nil; k, v = c.Next() {
				if !first {
					w.Write([]byte(","))
				}
				w.Write(v)
				first = false
			}
			return nil
		})
		if err != nil {
			log.Err(err).Msg("")
		}
		w.Write([]byte("]"))
	})

	return http.ListenAndServe(bindAddr, nil)
}
