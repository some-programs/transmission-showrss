package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/go-pa/fenv"
	"github.com/odwrtw/transmission"
	"github.com/some-programs/transmission-showrss/pkg/cmdline"
	"github.com/some-programs/transmission-showrss/pkg/log"
	"github.com/some-programs/transmission-showrss/pkg/showrss"
	"golang.org/x/sync/errgroup"
)

func main() {
	var (
		transmissionConfig = cmdline.TransmissionConfigFlags(flag.CommandLine)
		feedSelection      = cmdline.FeedSelectionFlags(flag.CommandLine)
		showDirs           = cmdline.ShowDirsFlags(flag.CommandLine)
	)

	var deleteAllTorrents bool
	// tf.Register(flag.CommandLine)
	flag.BoolVar(&deleteAllTorrents, "delete_all_torrents", false, "wipe torrent and torrent data from transmission")

	fenv.CommandLinePrefix("TMTOOL_")
	var logConfig log.Config
	logConfig.RegisterFlags(flag.CommandLine)

	fenv.MustParse()
	flag.Parse()

	logConfig.Setup()

	log.Info().Interface("feed_selection", feedSelection).Msg("config")
	if feedSelection.IsEmtpy() {
		fmt.Println("Must choose at least one user or show")
		os.Exit(1)
	}

	tc, err := transmission.New(*transmissionConfig)
	if err != nil {
		log.Fatal().Err(err).Msg("error creating transmission client")
	}

	if deleteAllTorrents {
		torrents, err := tc.GetTorrents()
		if err != nil {
			log.Fatal().Err(err).Msg("error getting list of torrents")
		}
		tc.RemoveTorrents(torrents, true)
	}

	db, err := showrss.NewDB("showrss.db")
	if err != nil {
		log.Fatal().Err(err).Msg("error connecting to database")
	}
	defer db.Close()

	downloader := showrss.ShowRSSDownloader{
		ShowDirs:  *showDirs,
		TC:        tc,
		DB:        db,
		Selection: *feedSelection,
	}

	ctx := context.Background()
	eg, ctx := errgroup.WithContext(ctx)

	eg.Go(func() error { return downloader.Start(ctx) })
	eg.Go(func() error { return showrss.APIServer(db, tc, ":8384") })

	if err := eg.Wait(); err != nil {
		log.Fatal().Err(err).Msg("exiting")
	}
}
