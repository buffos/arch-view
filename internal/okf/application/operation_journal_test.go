package application

import (
	"testing"

	"github.com/buffo/arch-view/internal/okf/domain"
)

func TestOperationJournalOwnsExactRetryResults(t *testing.T) {
	var journal operationJournal
	if _, err, found := journal.replay("missing", nil); err != nil || found {
		t.Fatal("empty journal replayed a result")
	}
	input := []byte("request")
	value := domain.ProjectConfiguration{Revision: "r1", Profiles: []domain.Profile{{ProfileID: "project:a", Name: "Original"}}}
	journal.record(" operation ", input, value)
	input[0] = 'x'
	value.Profiles[0].Name = "Mutated input"
	got, err, found := journal.replay("operation", []byte("request"))
	if err != nil || !found || got.Revision != "r1" || got.Profiles[0].Name != "Original" {
		t.Fatalf("retry=%+v err=%v found=%v", got, err, found)
	}
	got.Profiles[0].Name = "Mutated output"
	again, _, _ := journal.replay(" operation ", []byte("request"))
	if again.Profiles[0].Name != "Original" {
		t.Fatal("retry result aliases retained state")
	}
	if _, err, found := journal.replay("operation", []byte("changed")); !found || !hasApplicationCode(err, "okf_idempotency_conflict") {
		t.Fatalf("changed input accepted: %v", err)
	}
	journal.record(" ", nil, value)
	if _, err, found := journal.replay("", nil); err != nil || found {
		t.Fatal("blank operation was recorded")
	}
}
