package relay

import (
	"fmt"
	"testing"

	dbmodel "github.com/lingyuins/octopus/internal/model"
	"github.com/lingyuins/octopus/internal/relay/balancer"
)

func TestRecordPreparedCandidateSkip_DoesNotDuplicateCircuitBreak(t *testing.T) {
	modelName := fmt.Sprintf("gpt-4o-%s", t.Name())
	group := dbmodel.Group{
		Items: []dbmodel.GroupItem{
			{ChannelID: 11, ModelName: modelName},
		},
	}
	iter := balancer.NewIterator(group, 0, modelName)
	if !iter.Next() {
		t.Fatal("iterator should have one candidate")
	}

	for i := 0; i < 5; i++ {
		balancer.RecordFailure(11, 22, modelName)
	}
	if !iter.SkipCircuitBreak(11, 22, "test-channel", modelName) {
		t.Fatal("SkipCircuitBreak() = false, want true")
	}

	recordPreparedCandidateSkip(iter, iter.Item(), PrepareCandidateResult{
		Channel:    &dbmodel.Channel{ID: 11, Name: "test-channel"},
		UsedKey:    dbmodel.ChannelKey{ID: 22},
		SkipReason: "circuit breaker tripped",
		SkipStatus: dbmodel.AttemptCircuitBreak,
	})

	attempts := iter.Attempts()
	if len(attempts) != 1 {
		t.Fatalf("len(Attempts()) = %d, want 1", len(attempts))
	}
	if attempts[0].Status != dbmodel.AttemptCircuitBreak {
		t.Fatalf("Attempts()[0].Status = %s, want %s", attempts[0].Status, dbmodel.AttemptCircuitBreak)
	}
}

func TestRecordPreparedCandidateSkip_RecordsSkippedCandidate(t *testing.T) {
	group := dbmodel.Group{
		Items: []dbmodel.GroupItem{
			{ChannelID: 11, ModelName: "gpt-4o"},
		},
	}
	iter := balancer.NewIterator(group, 0, "gpt-4o")
	if !iter.Next() {
		t.Fatal("iterator should have one candidate")
	}

	recordPreparedCandidateSkip(iter, iter.Item(), PrepareCandidateResult{
		Channel:    &dbmodel.Channel{ID: 11, Name: "test-channel"},
		UsedKey:    dbmodel.ChannelKey{ID: 22},
		SkipReason: "no available key",
		SkipStatus: dbmodel.AttemptSkipped,
	})

	attempts := iter.Attempts()
	if len(attempts) != 1 {
		t.Fatalf("len(Attempts()) = %d, want 1", len(attempts))
	}
	if attempts[0].Status != dbmodel.AttemptSkipped {
		t.Fatalf("Attempts()[0].Status = %s, want %s", attempts[0].Status, dbmodel.AttemptSkipped)
	}
	if attempts[0].Msg != "no available key" {
		t.Fatalf("Attempts()[0].Msg = %q, want %q", attempts[0].Msg, "no available key")
	}
}
