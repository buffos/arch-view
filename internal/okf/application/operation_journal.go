package application

import (
	"bytes"
	"strings"
	"sync"

	"github.com/buffo/arch-view/internal/okf/domain"
)

// operationJournal retains successful configuration results for exact retries.
// It does not validate, persist, or publish configuration to sessions.
type operationJournal struct {
	mu      sync.RWMutex
	entries map[string]persistedOperation
}

type persistedOperation struct {
	input []byte
	value domain.ProjectConfiguration
}

func (journal *operationJournal) record(operationID string, input []byte, value domain.ProjectConfiguration) {
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return
	}
	journal.mu.Lock()
	defer journal.mu.Unlock()
	if journal.entries == nil {
		journal.entries = make(map[string]persistedOperation)
	}
	journal.entries[operationID] = persistedOperation{input: append([]byte(nil), input...), value: cloneConfiguration(value)}
}

func (journal *operationJournal) replay(operationID string, input []byte) (domain.ProjectConfiguration, error, bool) {
	operationID = strings.TrimSpace(operationID)
	if operationID == "" {
		return domain.ProjectConfiguration{}, nil, false
	}
	journal.mu.RLock()
	previous, exists := journal.entries[operationID]
	journal.mu.RUnlock()
	if !exists {
		return domain.ProjectConfiguration{}, nil, false
	}
	if !bytes.Equal(previous.input, input) {
		return domain.ProjectConfiguration{}, domain.NewError("okf_idempotency_conflict", 409, "operation ID was already used with different input", map[string]any{"operation_id": operationID}), true
	}
	return cloneConfiguration(previous.value), nil, true
}
