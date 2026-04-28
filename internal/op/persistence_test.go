package op

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/lingyuins/octopus/internal/db"
	"github.com/lingyuins/octopus/internal/model"
)

func TestChannelKeySaveDB_RequeuesDirtyIDsOnWriteFailure(t *testing.T) {
	restore := snapshotChannelKeyPersistenceState()
	defer restore()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(t.Name()))
	if err := db.InitDB("sqlite", dsn, false); err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	channelKeyCache.Set(7, model.ChannelKey{ID: 7, ChannelID: 3, Enabled: true, ChannelKey: "sk-test"})
	channelKeyCacheNeedUpdateLock.Lock()
	channelKeyCacheNeedUpdate[7] = struct{}{}
	channelKeyCacheNeedUpdateLock.Unlock()

	if err := db.Close(); err != nil {
		t.Fatalf("close db before failure simulation: %v", err)
	}

	if err := ChannelKeySaveDB(context.Background()); err == nil {
		t.Fatal("ChannelKeySaveDB() error = nil, want write failure")
	}

	channelKeyCacheNeedUpdateLock.Lock()
	_, ok := channelKeyCacheNeedUpdate[7]
	channelKeyCacheNeedUpdateLock.Unlock()
	if !ok {
		t.Fatal("dirty channel key id 7 was not requeued after write failure")
	}
}

func TestStatsSaveDB_RequeuesDirtyIDsOnWriteFailure(t *testing.T) {
	restore := snapshotStatsPersistenceState()
	defer restore()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(t.Name()))
	if err := db.InitDB("sqlite", dsn, false); err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	statsTotalCache = model.StatsTotal{ID: 1}
	statsDailyCache = model.StatsDaily{Date: time.Now().Format("20060102")}
	statsChannelCache.Set(11, model.StatsChannel{ChannelID: 11})
	statsModelCache.Set(22, model.StatsModel{ID: 22, Name: "gpt-4o", ChannelID: 11})
	statsAPIKeyCache.Set(33, model.StatsAPIKey{APIKeyID: 33})

	statsChannelCacheNeedUpdateLock.Lock()
	statsChannelCacheNeedUpdate[11] = struct{}{}
	statsChannelCacheNeedUpdateLock.Unlock()
	statsModelCacheNeedUpdateLock.Lock()
	statsModelCacheNeedUpdate[22] = struct{}{}
	statsModelCacheNeedUpdateLock.Unlock()
	statsAPIKeyCacheNeedUpdateLock.Lock()
	statsAPIKeyCacheNeedUpdate[33] = struct{}{}
	statsAPIKeyCacheNeedUpdateLock.Unlock()

	if err := db.Close(); err != nil {
		t.Fatalf("close db before failure simulation: %v", err)
	}

	if err := StatsSaveDB(context.Background()); err == nil {
		t.Fatal("StatsSaveDB() error = nil, want write failure")
	}

	statsChannelCacheNeedUpdateLock.Lock()
	_, channelDirty := statsChannelCacheNeedUpdate[11]
	statsChannelCacheNeedUpdateLock.Unlock()
	if !channelDirty {
		t.Fatal("channel dirty id 11 was not requeued after write failure")
	}

	statsModelCacheNeedUpdateLock.Lock()
	_, modelDirty := statsModelCacheNeedUpdate[22]
	statsModelCacheNeedUpdateLock.Unlock()
	if !modelDirty {
		t.Fatal("model dirty id 22 was not requeued after write failure")
	}

	statsAPIKeyCacheNeedUpdateLock.Lock()
	_, apiKeyDirty := statsAPIKeyCacheNeedUpdate[33]
	statsAPIKeyCacheNeedUpdateLock.Unlock()
	if !apiKeyDirty {
		t.Fatal("api key dirty id 33 was not requeued after write failure")
	}
}

func snapshotChannelKeyPersistenceState() func() {
	oldKeys := channelKeyCache.GetAll()
	channelKeyCacheNeedUpdateLock.Lock()
	oldDirty := make(map[int]struct{}, len(channelKeyCacheNeedUpdate))
	for id := range channelKeyCacheNeedUpdate {
		oldDirty[id] = struct{}{}
	}
	channelKeyCacheNeedUpdateLock.Unlock()

	return func() {
		channelKeyCache.Clear()
		for id, key := range oldKeys {
			channelKeyCache.Set(id, key)
		}

		channelKeyCacheNeedUpdateLock.Lock()
		channelKeyCacheNeedUpdate = make(map[int]struct{}, len(oldDirty))
		for id := range oldDirty {
			channelKeyCacheNeedUpdate[id] = struct{}{}
		}
		channelKeyCacheNeedUpdateLock.Unlock()
	}
}

func snapshotStatsPersistenceState() func() {
	oldTotal := statsTotalCache
	oldDaily := statsDailyCache
	oldHourly := statsHourlyCache
	oldChannelStats := statsChannelCache.GetAll()
	oldModelStats := statsModelCache.GetAll()
	oldAPIKeyStats := statsAPIKeyCache.GetAll()

	statsChannelCacheNeedUpdateLock.Lock()
	oldChannelDirty := make(map[int]struct{}, len(statsChannelCacheNeedUpdate))
	for id := range statsChannelCacheNeedUpdate {
		oldChannelDirty[id] = struct{}{}
	}
	statsChannelCacheNeedUpdateLock.Unlock()

	statsModelCacheNeedUpdateLock.Lock()
	oldModelDirty := make(map[int]struct{}, len(statsModelCacheNeedUpdate))
	for id := range statsModelCacheNeedUpdate {
		oldModelDirty[id] = struct{}{}
	}
	statsModelCacheNeedUpdateLock.Unlock()

	statsAPIKeyCacheNeedUpdateLock.Lock()
	oldAPIKeyDirty := make(map[int]struct{}, len(statsAPIKeyCacheNeedUpdate))
	for id := range statsAPIKeyCacheNeedUpdate {
		oldAPIKeyDirty[id] = struct{}{}
	}
	statsAPIKeyCacheNeedUpdateLock.Unlock()

	return func() {
		statsTotalCache = oldTotal
		statsDailyCache = oldDaily
		statsHourlyCache = oldHourly

		statsChannelCache.Clear()
		for id, stats := range oldChannelStats {
			statsChannelCache.Set(id, stats)
		}
		statsModelCache.Clear()
		for id, stats := range oldModelStats {
			statsModelCache.Set(id, stats)
		}
		statsAPIKeyCache.Clear()
		for id, stats := range oldAPIKeyStats {
			statsAPIKeyCache.Set(id, stats)
		}

		statsChannelCacheNeedUpdateLock.Lock()
		statsChannelCacheNeedUpdate = make(map[int]struct{}, len(oldChannelDirty))
		for id := range oldChannelDirty {
			statsChannelCacheNeedUpdate[id] = struct{}{}
		}
		statsChannelCacheNeedUpdateLock.Unlock()

		statsModelCacheNeedUpdateLock.Lock()
		statsModelCacheNeedUpdate = make(map[int]struct{}, len(oldModelDirty))
		for id := range oldModelDirty {
			statsModelCacheNeedUpdate[id] = struct{}{}
		}
		statsModelCacheNeedUpdateLock.Unlock()

		statsAPIKeyCacheNeedUpdateLock.Lock()
		statsAPIKeyCacheNeedUpdate = make(map[int]struct{}, len(oldAPIKeyDirty))
		for id := range oldAPIKeyDirty {
			statsAPIKeyCacheNeedUpdate[id] = struct{}{}
		}
		statsAPIKeyCacheNeedUpdateLock.Unlock()
	}
}
