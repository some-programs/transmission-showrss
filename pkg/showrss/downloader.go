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
	"github.com/pborzenkov/go-transmission/transmission"
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

	trDownloadDir      string
	sessionDownloadDir string

	started   bool
	newItemCh chan Episode // items coming from rss subscriptions

}

func (d *ShowRSSDownloader) Start(ctx context.Context) error {
	if d.started {
		return errors.New("already started")
	}
	d.started = true
	d.newItemCh = make(chan Episode)

	session, err := d.TC.GetSession(context.Background(), transmission.SessionFieldDownloadDirectory)
	if err != nil {
		return fmt.Errorf("error connecting to transmission: %v", err)
	}
	d.sessionDownloadDir = session.DownloadDirectory

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

	eg.Go(func() error { return d.handleItems(ctx) })

	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

func (d *ShowRSSDownloader) handleItems(ctx context.Context) error {
loop:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case item := <-d.newItemCh:
			logger := getLogger(item)
			logger.Debug().Msg("new item")
			err := d.DB.Update(func(tx *bolt.Tx) error {
				bucket := tx.Bucket(bucketAdded)
				valueData := bucket.Get(item.Key())
				found := valueData != nil
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
				if !found {
					logger.Debug().Msg("trying to add item to transmission")
					err := d.addTorrent(ctx, item)
					if err != nil {
						if err == errAlreadyAdded {
							logger.Debug().Msg("torrent already in transmission")
						} else {
							return fmt.Errorf("could not add torrent '%v' to transmission: %v", item, err)
						}
					} else {
						logger.Info().Msg("torrent added to transmission")
					}
				} else {
					logger.Info().Msg("item already in added db")
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
				logger.Err(err).Msg("error handling item")
				continue loop
			}
		}
	}
}

var errAlreadyAdded = errors.New("torrent already added")

func (d *ShowRSSDownloader) addTorrent(ctx context.Context, item Episode) error {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	logger := getLogger(item)

	infoHash := strings.ToLower(item.InfoHash)
	torrents, err := d.TC.GetTorrents(ctx,
		transmission.IDs(transmission.Hash(infoHash)),
		transmission.TorrentFieldName, transmission.TorrentFieldHash,
	)
	if err != err {
		return err
	}
	for _, v := range torrents {
		v := v
		logger := logger.With().Str("torrent_hash", string(v.Hash)).Logger()
		logger.Trace().Msg("compare torrent")
		if strings.ToLower(string(v.Hash)) == infoHash {
			logger.Info().Msg("already in transmission")
			return errAlreadyAdded
		}
	}
	var downloadDir string
	if d.ShowDirs.Path != "" {
		var root string
		if !filepath.IsAbs(d.ShowDirs.Path) {
			root = d.sessionDownloadDir
		}
		downloadDir = filepath.Clean(filepath.Join(root, d.ShowDirs.Path, item.ShowDirectoryName()))
	}
	_, err = d.TC.AddTorrent(context.Background(), &transmission.AddTorrentReq{
		DownloadDirectory: String(downloadDir),
		URL:               String(item.URL()),
	})
	if err != nil {
		logger.Err(err).Msg(spew.Sdump(torrents, item))
		return err
	}

	logger.Debug().Msg("added torrent")
	return nil
}

func String(s string) *string {
	return &s
}
