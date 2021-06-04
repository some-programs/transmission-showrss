package showrss

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/odwrtw/transmission"
	"github.com/rs/zerolog"
	"github.com/some-programs/transmission-showrss/pkg/log"
	bolt "go.etcd.io/bbolt"
	"golang.org/x/sync/errgroup"
)

func fmtLog(item Episode, msg string) string {
	return fmt.Sprintf("%s *** %s *** %s", item.InfoHash, msg, item.Title)
}

func getLogger(item Episode) zerolog.Logger {
	return log.With().Str("info_hash", item.InfoHash).Str("title", item.Title).Logger()
}

type FeedSelection struct {
	Shows []int
	Users []int
}

func (f FeedSelection) IsEmtpy() bool {
	return len(f.Users) == 0 && len(f.Shows) == 0
}

type ShowDirs struct {
	Path string
	Dirs bool
}

type ShowRSSDownloader struct {
	TC        *transmission.Client
	Selection FeedSelection
	ShowDirs  ShowDirs
	DB        *DB

	trDownloadDir string

	started     bool
	newItemCh   chan Episode // items coming from rss subscriptions
	addItemCh   chan Episode // items that should be added
	addedItemCh chan Episode // items that was added
}

func (d *ShowRSSDownloader) Start(ctx context.Context) error {
	if d.started {
		return errors.New("already started")
	}
	d.started = true
	d.newItemCh = make(chan Episode)
	d.addItemCh = make(chan Episode)
	d.addedItemCh = make(chan Episode, 100)

	if err := d.TC.Session.Update(); err != nil {
		return fmt.Errorf("error connecting to transmission: %v", err)
	}

	// show := showrss.NewClient(showrss.ClientTTL(time.Second))
	show := NewClient()

	eg, ctx := errgroup.WithContext(ctx)
	for _, userID := range d.Selection.Users {
		channel, err := show.GetUserFeed(ctx, userID)
		if err != nil {
			return fmt.Errorf("error during initial fetch of user channel %v: %v", userID, err)
		}
		log.Info().
			Int("user_id", userID).
			Str("title", channel.Title).
			Msg("adding monitor for user")
		monitorFunc := func(channel Channel) func() error {
			return func() error {
				return show.MonitorChannel(ctx, channel, d.newItemCh)
			}
		}
		eg.Go(monitorFunc(*channel))
	}

	for _, showID := range d.Selection.Shows {
		channel, err := show.GetShowFeed(ctx, showID)
		if err != nil {
			return fmt.Errorf("error during initial fetch of show channel %v: %v", showID, err)
		}
		log.Info().
			Int("show_id", showID).
			Str("title", channel.Title).
			Msg("adding monitor for show")
		monitorFunc := func(channel Channel) func() error {
			return func() error {
				return show.MonitorChannel(ctx, channel, d.newItemCh)
			}
		}
		eg.Go(monitorFunc(*channel))
	}

	eg.Go(func() error { return d.filterTorrents(ctx) })
	eg.Go(func() error { return d.addedTorrents(ctx) })
	eg.Go(func() error { return d.addTorrents(ctx) })
	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

func (d *ShowRSSDownloader) filterTorrents(ctx context.Context) error {
loop:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case item := <-d.newItemCh:
			logger := getLogger(item)
			logger.Debug().Msg("new item")
			var found bool
			err := d.DB.Update(func(tx *bolt.Tx) error {
				bucket := tx.Bucket(bucketAdded)
				valueData := bucket.Get(item.Key())
				if valueData != nil {
					found = true
				}
				var dbep dbEpisode
				if found {
					if err := json.Unmarshal(valueData, &dbep); err != nil {
						return err
					}
					dbep.Updated = time.Now()
				} else {
					var err error
					dbep, err = newDBEpisode(item)
					if err != nil {
						return err
					}
				}
				data, err := json.Marshal(&dbep)
				if err != nil {
					return err
				}
				if err := bucket.Put(item.Key(), data); err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				logger.Err(err).Msg("")
				continue loop
			}
			if !found {
				d.addItemCh <- item
			} else {
				logger.Debug().Msg("item already in added db")
			}
		}
	}
}

func (d *ShowRSSDownloader) addedTorrents(ctx context.Context) error {
	for item := range d.addedItemCh {
		logger := getLogger(item)
		logger.Info().Msg("addedTorrents")
		err := d.DB.Update(func(tx *bolt.Tx) error {
			bucket := tx.Bucket(bucketAdded)
			v, err := newDBEpisode(item)
			if err != nil {
				logger.Fatal().Err(err).Msg("")
			}
			value, err := json.Marshal(&v)
			if err != nil {
				logger.Err(err).Msg("")
				return err
			}
			err = bucket.Put(item.Key(), value)
			if err != nil {
				return err
			}
			logger.Debug().Msgf("saved to db: %s:  %s", string(item.Key()), string(value))
			return nil
		})
		if err != nil {
			logger.Err(err).Msg("")
		}
	}
	return nil
}

func (d *ShowRSSDownloader) addTorrents(ctx context.Context) error {
items:
	for item := range d.addItemCh {
		logger := getLogger(item)
		logger.Info().Msg("addTorrents")
		err := d.addTorrent(item)
		if err != nil {
			if err == errAlreadyAdded {
				d.addedItemCh <- item
				continue items
			}
			logger.Err(err).Msg("")
			continue items
		}
		logger.Info().Msg("added torrent")
		d.addedItemCh <- item
	}
	return nil
}

var errAlreadyAdded = errors.New("torrent already added")

func (d *ShowRSSDownloader) addTorrent(item Episode) error {
	logger := getLogger(item)

	// Get all torrents
	torrents, err := d.TC.GetTorrents()
	if err != err {
		return err
	}
	for _, v := range torrents {
		if strings.ToLower(v.HashString) == strings.ToLower(item.InfoHash) {

			logger.Info().Msg("already in transmission")
			return errAlreadyAdded
		}
	}
	var downloadDir string
	if d.ShowDirs.Path != "" {
		var root string
		if !filepath.IsAbs(d.ShowDirs.Path) {
			root = d.TC.Session.DownloadDir
		}
		downloadDir = filepath.Clean(filepath.Join(root, d.ShowDirs.Path, item.ShowDirectoryName()))
	}
	_, err = d.TC.AddTorrent(transmission.AddTorrentArg{
		DownloadDir: downloadDir,
		Filename:    item.URL(),
	})

	if err != nil {
		logger.Err(err).Msg(spew.Sdump(torrents, item))
		return err
	}
	return nil
}
