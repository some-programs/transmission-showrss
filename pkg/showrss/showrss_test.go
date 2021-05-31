package showrss

import (
	"testing"
	"time"
)

func makeEpisodes(strs ...string) []Episode {
	var eps []Episode
	for _, s := range strs {
		eps = append(eps, Episode{InfoHash: s})
	}
	return eps
}

func TestFeature(t *testing.T) {
	assertKeys := func(dc *dedupCache, keys ...string) {
		if len(keys) != len(dc.items) {
			t.Errorf("not expteded length: %v %v", keys, dc)
		}
		for _, k := range keys {
			if _, ok := dc.items[k]; !ok {
				t.Errorf("could not find %s in %v", k, dc.items)
			}
		}
	}

	{
		dc := &dedupCache{
			items: make(map[string]time.Time),
		}
		for _, v := range []string{"1", "2", "3", "4", "5", "6"} {
			dc.Update(makeEpisodes(v))
		}
		assertKeys(dc, "4", "5", "6")
		for _, v := range []string{"7", "8"} {
			dc.Update(makeEpisodes(v))
		}
		assertKeys(dc, "7", "8")

		for _, v := range []string{"7", "8", "8", "7", "8", "8"} {
			dc.Update(makeEpisodes(v))
		}
		assertKeys(dc, "7", "8")

		dc.clean()
		assertKeys(dc, "8")

	}
	{
		dc := &dedupCache{
			items: make(map[string]time.Time),
		}
		for _, v := range [][]string{{"1", "2"}, {"3", "4"}, {"5", "6"}} {
			dc.Update(makeEpisodes(v...))
		}
		assertKeys(dc, "1", "2", "3", "4", "5", "6")

		dc.Update(makeEpisodes("7", "8"))
		assertKeys(dc, "7", "8")

		dc.clean()
		assertKeys(dc, "7", "8")

	}
}
