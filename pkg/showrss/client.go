package showrss

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/some-programs/transmission-showrss/pkg/log"
)

var defaultHTTPClient = &http.Client{
	Transport: &http.Transport{
		ExpectContinueTimeout: time.Minute,
		ResponseHeaderTimeout: time.Minute,
		TLSHandshakeTimeout:   time.Minute,
	},
	Timeout: time.Minute,
}

type clientOpt func(c *Client) error

func ClientTTL(ttl time.Duration) clientOpt {
	return func(c *Client) error {
		c.ttl = &ttl
		return nil
	}
}

func ClientHTTPClint(c *http.Client) clientOpt {
	return func(sc *Client) error {
		sc.httpClient = c
		return nil
	}
}

func NewClient(opts ...clientOpt) *Client {
	c := &Client{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

const defaultBaseURL = "http://showrss.info"

// Client .
type Client struct {
	httpClient *http.Client
	ttl        *time.Duration
	baseURL    string
}

func (c *Client) makeURL(path string) string {
	baseURL := c.baseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return fmt.Sprintf("%v%v", baseURL, path)
}

func (c *Client) GetUserFeed(ctx context.Context, ID int) (*Channel, error) {
	return c.get(ctx, c.makeURL(fmt.Sprintf("/user/%v.rss?magnets=true&namespaces=true&name=clean&quality=null&re=yes", ID)))
}

func (c *Client) GetShowFeed(ctx context.Context, ID int) (*Channel, error) {
	return c.get(ctx, c.makeURL(fmt.Sprintf("/show/%v.rss?magnets=true&namespaces=true&name=clean&quality=fhd&re=yes", ID)))
}

func (c *Client) get(ctx context.Context, url string) (*Channel, error) {
	httpClient := c.httpClient
	if httpClient == nil {
		httpClient = defaultHTTPClient
	}
	log.Info().Str("feed_url", url).Msg("")
	var cancel context.CancelFunc
	var getCtx context.Context
	getCtx, cancel = context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(getCtx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	channel, err := ParseRSS(data)
	if err != nil {
		return nil, err
	}
	channel.URL = url
	return channel, nil
}

func (c *Client) MonitorChannel(ctx context.Context, channel Channel, episodeCh chan<- Episode) error {
	var ttl time.Duration
	if c.ttl != nil {
		ttl = *c.ttl
	} else {
		ttl = channel.TTLDuration()
	}
	log.Debug().Msgf("tt: %v", ttl)

	var currentChannel *Channel

	var last time.Time
	var failures int
	dc := &dedupCache{
		items: make(map[string]time.Time),
	}
	var bo *backoff.ExponentialBackOff
	currentChannel = &channel
loop:
	for {
		if currentChannel == nil {
			log.Debug().Msgf("get channel.URL: %v", channel.URL)
			var err error
			currentChannel, err = c.get(context.Background(), channel.URL)
			if err != nil {
				if bo == nil {
					bo = &backoff.ExponentialBackOff{
						InitialInterval:     2 * time.Second,
						RandomizationFactor: 0.5,
						Multiplier:          1.5,
						MaxInterval:         time.Hour,
						MaxElapsedTime:      0, // never stop
						Clock:               backoff.SystemClock,
					}
					bo.Reset()
				}
				failures++
				next := bo.NextBackOff()
				log.Warn().Msgf("failures: %v next:%v err: %v", failures, next, err)
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(next):
					continue loop
				}
			}
		}
		bo = nil
		failures = 0
		last = time.Now()
		if currentChannel != nil {
			epslen := len(currentChannel.Episodes)

			log.Debug().Msgf("episodes: %v", epslen)
			if currentChannel.Episodes != nil {
			eps:
				for _, item := range currentChannel.Episodes {
					if dc.Has(item) {
						log.Debug().Interface("item", item).Msg("already sent")
						continue eps
					}
					select {
					case <-ctx.Done():
						return ctx.Err()
					case episodeCh <- item:
						log.Debug().Interface("item", item).Msg("sent")
					}
				}
				dc.Update(currentChannel.Episodes)
			}
			currentChannel = nil
		}
		waitFor := ttl - time.Since(last)
		log.Info().Msgf("will wait for %v", waitFor)
		if waitFor > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitFor):
				continue loop
			}
		}
	}
}

// dedupCache .
type dedupCache struct {
	items  map[string]time.Time
	maxEps int
}

func (d *dedupCache) Has(e Episode) bool {
	_, ok := d.items[e.InfoHash]
	return ok
}

func (d *dedupCache) Update(eps []Episode) {
	l := len(eps)
	if l > d.maxEps {
		d.maxEps = l
	}
	if l > 0 {
		now := time.Now()
		for _, ep := range eps {
			d.items[ep.InfoHash] = now
		}
	}
	if len(d.items) > 3*d.maxEps {
		d.clean()
	}
}

func (d *dedupCache) clean() {
	type kv struct {
		k string
		v time.Time
	}
	var ss []kv
	for k, v := range d.items {
		ss = append(ss, kv{k, v})
	}
	sort.Slice(
		ss, func(i, j int) bool {
			return ss[i].v.After(ss[j].v)
		})
	for _, mk := range ss[d.maxEps:] {
		delete(d.items, mk.k)
	}
}
