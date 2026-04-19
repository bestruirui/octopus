package op

import (
	"context"
	"testing"

	"github.com/lingyuins/octopus/internal/db"
	"github.com/lingyuins/octopus/internal/model"
)

func TestGroupUpdateNormalizesItemsToAdd(t *testing.T) {
	ctx := context.Background()
	if err := db.InitDB("sqlite", "file:group-update-normalizes-items-to-add?mode=memory&cache=shared", false); err != nil {
		t.Fatalf("init db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
		groupCache.Clear()
		groupMap.Clear()
		rebuildGroupIndexesFromCache()
	})

	groupCache.Clear()
	groupMap.Clear()
	rebuildGroupIndexesFromCache()

	group := &model.Group{
		Name:         "test-group-update-normalize",
		EndpointType: model.EndpointTypeChat,
		Mode:         model.GroupModeRoundRobin,
		Items: []model.GroupItem{
			{ChannelID: 1, ModelName: "gpt-4o-mini", Priority: 1, Weight: 2},
			{ChannelID: 2, ModelName: "claude-3-5-haiku", Priority: 2, Weight: 3},
		},
	}
	if err := GroupCreate(group, ctx); err != nil {
		t.Fatalf("create group: %v", err)
	}

	itemsBeforeUpdate, err := GroupItemList(group.ID, ctx)
	if err != nil {
		t.Fatalf("list group items before update: %v", err)
	}
	if len(itemsBeforeUpdate) != 2 {
		t.Fatalf("expected 2 initial items, got %d", len(itemsBeforeUpdate))
	}

	req := &model.GroupUpdateRequest{
		ID: group.ID,
		ItemsToUpdate: []model.GroupItemUpdateRequest{
			{ID: itemsBeforeUpdate[0].ID, Priority: 1, Weight: 0},
			{ID: itemsBeforeUpdate[1].ID, Priority: 4, Weight: 6},
		},
		ItemsToAdd: []model.GroupItemAddRequest{
			{ChannelID: 1, ModelName: "gpt-4o-mini", Priority: 9, Weight: 9},
			{ChannelID: 3, ModelName: " claude-3-5-sonnet ", Priority: 3, Weight: 0},
			{ChannelID: 4, ModelName: "  ", Priority: 5, Weight: 2},
			{ChannelID: 5, ModelName: "gpt-4.1-mini", Priority: 0, Weight: 0},
			{ChannelID: 3, ModelName: "claude-3-5-sonnet", Priority: 7, Weight: 3},
		},
	}

	updated, err := GroupUpdate(req, ctx)
	if err != nil {
		t.Fatalf("update group: %v", err)
	}

	if len(updated.Items) != 4 {
		t.Fatalf("expected 4 items after update, got %d", len(updated.Items))
	}

	itemsAfterUpdate, err := GroupItemList(group.ID, ctx)
	if err != nil {
		t.Fatalf("list group items after update: %v", err)
	}
	if len(itemsAfterUpdate) != 4 {
		t.Fatalf("expected 4 persisted items after update, got %d", len(itemsAfterUpdate))
	}

	if itemsAfterUpdate[0].ChannelID != 1 || itemsAfterUpdate[0].ModelName != "gpt-4o-mini" || itemsAfterUpdate[0].Priority != 1 || itemsAfterUpdate[0].Weight != 2 {
		t.Fatalf("unexpected first persisted item: %+v", itemsAfterUpdate[0])
	}
	if itemsAfterUpdate[1].ChannelID != 3 || itemsAfterUpdate[1].ModelName != "claude-3-5-sonnet" || itemsAfterUpdate[1].Priority != 3 || itemsAfterUpdate[1].Weight != 1 {
		t.Fatalf("unexpected second persisted item: %+v", itemsAfterUpdate[1])
	}
	if itemsAfterUpdate[2].ChannelID != 2 || itemsAfterUpdate[2].ModelName != "claude-3-5-haiku" || itemsAfterUpdate[2].Priority != 4 || itemsAfterUpdate[2].Weight != 6 {
		t.Fatalf("unexpected third persisted item: %+v", itemsAfterUpdate[2])
	}
	if itemsAfterUpdate[3].ChannelID != 5 || itemsAfterUpdate[3].ModelName != "gpt-4.1-mini" || itemsAfterUpdate[3].Priority != 5 || itemsAfterUpdate[3].Weight != 1 {
		t.Fatalf("unexpected fourth persisted item: %+v", itemsAfterUpdate[3])
	}
}
