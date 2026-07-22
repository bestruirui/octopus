package op

import (
	"context"
	"slices"
	"testing"

	"github.com/bestruirui/octopus/internal/model"
)

func TestGroupListModelReturnsSortedNames(t *testing.T) {
	// Snapshot and restore package cache so this test does not leak into others.
	prev := groupCache.GetAll()
	t.Cleanup(func() {
		groupCache.Clear()
		for id, group := range prev {
			groupCache.Set(id, group)
		}
	})

	groupCache.Clear()
	// Insert in non-sorted order to prove ordering comes from GroupListModel.
	groupCache.Set(3, model.Group{ID: 3, Name: "zeta-model"})
	groupCache.Set(1, model.Group{ID: 1, Name: "alpha-model"})
	groupCache.Set(2, model.Group{ID: 2, Name: "mid-model"})

	got, err := GroupListModel(context.Background())
	if err != nil {
		t.Fatalf("GroupListModel returned error: %v", err)
	}

	want := []string{"alpha-model", "mid-model", "zeta-model"}
	if !slices.Equal(got, want) {
		t.Fatalf("GroupListModel order = %v, want %v", got, want)
	}
}
