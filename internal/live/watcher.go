package live

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var validWatchKinds = map[string]struct{}{
	"create": {}, "modify": {}, "delete": {}, "rename": {}, "overflow": {}, "error": {},
}

// NormalizeWatchEvent converts a backend event into the repository-relative
// form consumed by the coordinator. It accepts absolute backend paths only
// when they resolve beneath the configured repository root.
func NormalizeWatchEvent(event WatchEvent, repositoryRoot string, roots []WatchRoot) (NormalizedEvent, error) {
	kind := strings.ToLower(strings.TrimSpace(event.Kind))
	if _, ok := validWatchKinds[kind]; !ok {
		return NormalizedEvent{}, newLiveError("WatchEventInvalid", "watch event kind is unsupported", map[string]any{"kind": event.Kind})
	}
	normalized := NormalizedEvent{BackendID: strings.TrimSpace(event.BackendID), Kind: kind, Sequence: strings.TrimSpace(event.Sequence)}
	if kind != "overflow" && kind != "error" || strings.TrimSpace(event.Path) != "" {
		pathValue, err := normalizeWatchEventPath(event.Path, repositoryRoot, roots)
		if err != nil {
			return NormalizedEvent{}, err
		}
		normalized.Path = pathValue
	}
	if strings.TrimSpace(event.OldPath) != "" {
		oldPath, err := normalizeWatchEventPath(event.OldPath, repositoryRoot, roots)
		if err != nil {
			return NormalizedEvent{}, newLiveError("WatchEventOutOfScope", "watch rename old_path is outside a configured watch root", map[string]any{"path": event.OldPath})
		}
		normalized.OldPath = oldPath
	}
	if normalized.Path == "" && normalized.OldPath == "" && kind != "overflow" && kind != "error" {
		return NormalizedEvent{}, newLiveError("WatchEventInvalid", "watch event requires a path", nil)
	}
	normalized.EventGroupID = eventIdentity(normalized)
	return normalized, nil
}

func normalizeWatchEventPath(value, repositoryRoot string, roots []WatchRoot) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	root, err := filepath.Abs(repositoryRoot)
	if err != nil {
		return "", newLiveError("WatchEventOutOfScope", "repository root could not be normalized", nil)
	}
	var relative string
	if filepath.IsAbs(value) {
		relative, err = filepath.Rel(root, filepath.Clean(value))
		if err != nil {
			return "", newLiveError("WatchEventOutOfScope", "watch path could not be made repository-relative", map[string]any{"path": value})
		}
	} else {
		relative = filepath.ToSlash(filepath.Clean(strings.ReplaceAll(value, "\\", "/")))
	}
	relative = strings.TrimPrefix(filepath.ToSlash(relative), "./")
	if relative == "" || relative == "." || relative == ".." || strings.HasPrefix(relative, "../") || strings.Contains(relative, "\x00") || filepath.VolumeName(relative) != "" {
		return "", newLiveError("WatchEventOutOfScope", "watch path escapes the repository root", map[string]any{"path": value})
	}
	if len(roots) == 0 {
		roots = []WatchRoot{{Path: ".", Recursive: true}}
	}
	allowed := false
	for _, watchRoot := range roots {
		watchPath := strings.TrimPrefix(filepath.ToSlash(filepath.Clean(watchRoot.Path)), "./")
		if watchPath == "." || watchPath == "" {
			allowed = true
		} else if relative == watchPath || strings.HasPrefix(relative, watchPath+"/") {
			allowed = true
		}
		if allowed && !watchRoot.Recursive && watchPath != "." && strings.Contains(strings.TrimPrefix(strings.TrimPrefix(relative, watchPath), "/"), "/") {
			allowed = false
		}
		if allowed {
			break
		}
	}
	if !allowed {
		return "", newLiveError("WatchEventOutOfScope", "watch path is outside configured watch roots", map[string]any{"path": relative})
	}
	candidate := filepath.Clean(filepath.Join(root, filepath.FromSlash(relative)))
	if !eventPathResolvesInside(root, candidate) {
		return "", newLiveError("WatchEventOutOfScope", "watch path resolves outside the repository root", map[string]any{"path": relative})
	}
	return relative, nil
}

// Deleted paths cannot always be resolved themselves. Walk up to the nearest
// existing ancestor so a symlinked directory cannot smuggle an out-of-root
// event through a lexically safe path.
func eventPathResolvesInside(root, candidate string) bool {
	root = filepath.Clean(root)
	probe := filepath.Clean(candidate)
	for pathWithin(root, probe) {
		if _, err := os.Lstat(probe); err == nil {
			resolved, resolveErr := filepath.EvalSymlinks(probe)
			return resolveErr == nil && pathWithin(root, resolved)
		}
		parent := filepath.Dir(probe)
		if parent == probe {
			break
		}
		probe = parent
	}
	return pathWithin(root, candidate)
}

func eventIdentity(event NormalizedEvent) string {
	data, _ := json.Marshal(struct {
		BackendID string `json:"backend_id"`
		Kind      string `json:"kind"`
		Path      string `json:"path"`
		OldPath   string `json:"old_path"`
		Sequence  string `json:"sequence"`
	}{event.BackendID, event.Kind, event.Path, event.OldPath, event.Sequence})
	hash := sha256.Sum256(data)
	return "event-" + hex.EncodeToString(hash[:8])
}

// EventCoalescer is a deterministic, bounded event reducer. Time is supplied
// by the caller, so tests can flush a debounce window without sleeping.
type EventCoalescer struct {
	mu           sync.Mutex
	policy       WatchPolicy
	pending      map[string]NormalizedEvent
	fullRescan   bool
	reason       string
	lastEventAt  time.Time
	lastSequence int
	sequenceSeen bool
}

func NewEventCoalescer(policy WatchPolicy) *EventCoalescer {
	if policy.MaxPendingEvents <= 0 {
		policy.MaxPendingEvents = DefaultMaxPendingEvents
	}
	return &EventCoalescer{policy: policy, pending: make(map[string]NormalizedEvent)}
}

func (coalescer *EventCoalescer) Add(event NormalizedEvent, now time.Time) {
	if coalescer == nil {
		return
	}
	coalescer.mu.Lock()
	defer coalescer.mu.Unlock()
	coalescer.lastEventAt = now
	if sequence, err := strconv.Atoi(strings.TrimSpace(event.Sequence)); err == nil && sequence > 0 {
		// Duplicate notifications are normal for several watcher backends. A
		// gap is only proven when a newer event skips a sequence number; an
		// older or duplicate notification must not force a full rescan.
		if coalescer.sequenceSeen && sequence > coalescer.lastSequence+1 {
			coalescer.fullRescan = true
			if coalescer.reason == "" {
				coalescer.reason = "watch_sequence_gap"
			}
		}
		if !coalescer.sequenceSeen || sequence > coalescer.lastSequence {
			coalescer.lastSequence = sequence
			coalescer.sequenceSeen = true
		}
	}
	if event.Kind == "overflow" || event.Kind == "error" {
		coalescer.fullRescan = true
		if coalescer.reason == "" {
			coalescer.reason = event.Kind
		}
		// The diagnostic itself is represented by the full-rescan reason. Do
		// not retain an unbounded stream of backend error/overflow records.
		return
	}
	if event.Kind == "rename" && (event.Path == "" || event.OldPath == "") {
		coalescer.fullRescan = true
		if coalescer.reason == "" {
			coalescer.reason = "ambiguous_rename"
		}
	}
	key := eventKey(event)
	if event.OldPath != "" {
		key = event.OldPath + "\x00" + event.Path
	}
	if previous, exists := coalescer.pending[key]; exists {
		merged, keep := mergeEvents(previous, event)
		if keep {
			coalescer.pending[key] = merged
		} else {
			delete(coalescer.pending, key)
		}
		return
	}
	if len(coalescer.pending) >= coalescer.policy.MaxPendingEvents {
		coalescer.pending = make(map[string]NormalizedEvent)
		coalescer.fullRescan = true
		coalescer.reason = "pending_event_limit_exceeded"
		return
	}
	coalescer.pending[key] = event
}

func (coalescer *EventCoalescer) Ready(now time.Time) bool {
	if coalescer == nil {
		return false
	}
	coalescer.mu.Lock()
	defer coalescer.mu.Unlock()
	if len(coalescer.pending) == 0 && !coalescer.fullRescan {
		return false
	}
	if coalescer.policy.DebounceMS <= 0 || coalescer.lastEventAt.IsZero() {
		return true
	}
	return !now.Before(coalescer.lastEventAt.Add(time.Duration(coalescer.policy.DebounceMS) * time.Millisecond))
}

func (coalescer *EventCoalescer) Flush() *EventGroup {
	if coalescer == nil {
		return nil
	}
	coalescer.mu.Lock()
	defer coalescer.mu.Unlock()
	return coalescer.flushLocked()
}

func (coalescer *EventCoalescer) FlushIfReady(now time.Time) *EventGroup {
	if !coalescer.Ready(now) {
		return nil
	}
	return coalescer.Flush()
}

func (coalescer *EventCoalescer) flushLocked() *EventGroup {
	if len(coalescer.pending) == 0 && !coalescer.fullRescan {
		return nil
	}
	events := make([]NormalizedEvent, 0, len(coalescer.pending))
	for _, event := range coalescer.pending {
		events = append(events, event)
	}
	sort.Slice(events, func(i, j int) bool {
		left := events[i].Path + "\x00" + events[i].OldPath + "\x00" + events[i].Kind + "\x00" + events[i].Sequence
		right := events[j].Path + "\x00" + events[j].OldPath + "\x00" + events[j].Kind + "\x00" + events[j].Sequence
		return left < right
	})
	paths := make(map[string]struct{})
	first, last := "", ""
	groupID := groupIdentity(events)
	if len(events) == 0 {
		groupID = rescanIdentity(coalescer.reason)
	}
	for index := range events {
		events[index].EventGroupID = groupID
		if events[index].Path != "" {
			paths[events[index].Path] = struct{}{}
		}
		if events[index].OldPath != "" {
			paths[events[index].OldPath] = struct{}{}
		}
		if events[index].Sequence != "" {
			if first == "" || events[index].Sequence < first {
				first = events[index].Sequence
			}
			if events[index].Sequence > last {
				last = events[index].Sequence
			}
		}
	}
	changed := make([]string, 0, len(paths))
	for path := range paths {
		changed = append(changed, path)
	}
	sort.Strings(changed)
	group := &EventGroup{ID: groupID, Events: events, ChangedPaths: changed, FullRescan: coalescer.fullRescan, Reason: coalescer.reason, FirstSequence: first, LastSequence: last}
	coalescer.pending = make(map[string]NormalizedEvent)
	coalescer.fullRescan = false
	coalescer.reason = ""
	coalescer.lastEventAt = time.Time{}
	return group
}

func rescanIdentity(reason string) string {
	data, _ := json.Marshal(struct {
		Kind   string `json:"kind"`
		Reason string `json:"reason"`
	}{Kind: "full_rescan", Reason: reason})
	hash := sha256.Sum256(data)
	return "event-group-rescan-" + hex.EncodeToString(hash[:8])
}

func mergeEvents(previous, current NormalizedEvent) (NormalizedEvent, bool) {
	merged := current
	merged.EventGroupID = previous.EventGroupID
	switch {
	case previous.Kind == "create" && current.Kind == "modify":
		merged.Kind = "create"
	case previous.Kind == "create" && current.Kind == "delete":
		return NormalizedEvent{}, false
	case previous.Kind == "modify" && current.Kind == "modify":
		merged.Kind = "modify"
	case previous.Kind == "delete" && current.Kind == "create":
		merged.Kind = "modify"
	case previous.Kind == "rename" || current.Kind == "rename":
		merged.Kind = "rename"
		if merged.OldPath == "" {
			merged.OldPath = previous.OldPath
		}
	default:
		merged.Kind = current.Kind
	}
	return merged, true
}

func eventKey(event NormalizedEvent) string {
	if event.Path != "" {
		return event.Path
	}
	return event.Kind + "\x00" + event.Sequence
}

func groupIdentity(events []NormalizedEvent) string {
	data, _ := json.Marshal(events)
	hash := sha256.Sum256(data)
	return "event-group-" + hex.EncodeToString(hash[:8])
}

// PlanInvalidation turns watcher evidence into a conservative rebuild plan.
// Selective invalidation is returned only when every changed path maps to one
// explicit scope root. The default session scanner may still choose a full
// scan; the plan records why it did so.
func PlanInvalidation(group EventGroup, scopes []ScopeIdentity) InvalidationPlan {
	plan := InvalidationPlan{EventGroupID: group.ID, AffectedPaths: append([]string(nil), group.ChangedPaths...), AffectedScopeIDs: []string{}, Mode: "full_rescan", Reason: group.Reason}
	if group.FullRescan {
		if plan.Reason == "" {
			plan.Reason = "watcher_requested_full_rescan"
		}
		return plan
	}
	matched := make(map[string]struct{})
	for _, changedPath := range group.ChangedPaths {
		pathMatched := []string{}
		for _, scope := range scopes {
			root := strings.TrimPrefix(filepath.ToSlash(filepath.Clean(scope.ProjectRoot)), "./")
			if root == "." || root == "" || changedPath == root || strings.HasPrefix(changedPath, root+"/") {
				pathMatched = append(pathMatched, scope.ScopeID)
			}
		}
		if len(pathMatched) != 1 {
			plan.Reason = "scope_or_dependency_impact_uncertain"
			return plan
		}
		matched[pathMatched[0]] = struct{}{}
	}
	if len(matched) == 0 {
		plan.Reason = "no_explicit_scope_match"
		return plan
	}
	for scopeID := range matched {
		plan.AffectedScopeIDs = append(plan.AffectedScopeIDs, scopeID)
	}
	sort.Strings(plan.AffectedScopeIDs)
	plan.Mode = "selective"
	if plan.Reason == "" {
		plan.Reason = "explicit_scope_root_match"
	}
	return plan
}

// FakeWatchBackend is deterministic and injectable. Tests push events through
// Emit rather than waiting for an operating-system watcher.
type FakeWatchBackend struct {
	mu      sync.Mutex
	channel chan WatchEvent
	started bool
}

func NewFakeWatchBackend(buffer int) *FakeWatchBackend {
	if buffer < 1 {
		buffer = 32
	}
	return &FakeWatchBackend{channel: make(chan WatchEvent, buffer)}
}

func (backend *FakeWatchBackend) Start(context.Context, string, []WatchRoot, WatchPolicy) (<-chan WatchEvent, error) {
	if backend == nil {
		return nil, newLiveError(ErrorWatchBackendUnavailable, "fake watcher is nil", nil)
	}
	backend.mu.Lock()
	backend.started = true
	backend.mu.Unlock()
	return backend.channel, nil
}

func (backend *FakeWatchBackend) Emit(event WatchEvent) error {
	if backend == nil {
		return newLiveError(ErrorWatchBackendUnavailable, "fake watcher is nil", nil)
	}
	backend.mu.Lock()
	started := backend.started
	backend.mu.Unlock()
	if !started {
		return newLiveError(ErrorWatchBackendUnavailable, "fake watcher has not started", nil)
	}
	select {
	case backend.channel <- event:
		return nil
	default:
		return newLiveError("WatchEventOverflow", "fake watcher event buffer is full", nil)
	}
}

func (backend *FakeWatchBackend) Stop() error {
	if backend == nil {
		return nil
	}
	backend.mu.Lock()
	backend.started = false
	backend.mu.Unlock()
	return nil
}

// LocalWatchBackend is a portable polling watcher. Polling is intentional:
// the watcher is a replaceable change-hint source, while authoritative
// freshness still comes from FileSystemFingerprinter.
type LocalWatchBackend struct {
	mu            sync.Mutex
	cancel        context.CancelFunc
	started       bool
	fingerprinter Fingerprinter
}

func NewLocalWatchBackend(fingerprinter Fingerprinter) *LocalWatchBackend {
	if fingerprinter == nil {
		fingerprinter = FileSystemFingerprinter{}
	}
	return &LocalWatchBackend{fingerprinter: fingerprinter}
}

func (backend *LocalWatchBackend) Start(ctx context.Context, repositoryRoot string, roots []WatchRoot, policy WatchPolicy) (<-chan WatchEvent, error) {
	if backend == nil {
		return nil, newLiveError(ErrorWatchBackendUnavailable, "local watcher is nil", nil)
	}
	if backend.fingerprinter == nil {
		backend.fingerprinter = FileSystemFingerprinter{}
	}
	if ctx == nil {
		ctx = context.Background()
	}
	interval := time.Duration(policy.RescanIntervalMS) * time.Millisecond
	if interval <= 0 {
		interval = time.Second
	}
	baseline, err := backend.fingerprinter.Fingerprint(ctx, repositoryRoot, roots)
	if err != nil {
		return nil, err
	}
	watchCtx, cancel := context.WithCancel(ctx)
	backend.mu.Lock()
	if backend.cancel != nil {
		backend.cancel()
	}
	backend.cancel = cancel
	backend.started = true
	backend.mu.Unlock()
	output := make(chan WatchEvent, 64)
	go func() {
		defer close(output)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		previous := baseline
		sequence := 0
		for {
			select {
			case <-watchCtx.Done():
				return
			case <-ticker.C:
				current, fingerprintErr := backend.fingerprinter.Fingerprint(watchCtx, repositoryRoot, roots)
				if fingerprintErr != nil {
					select {
					case output <- WatchEvent{BackendID: "watch:polling", Kind: "error", Sequence: fmt.Sprintf("%d", sequence)}:
					case <-watchCtx.Done():
						return
					}
					continue
				}
				for _, path := range changedFingerprintPaths(previous, current) {
					sequence++
					kind := "modify"
					if !containsFingerprintPath(previous.Files, path) {
						kind = "create"
					} else if !containsFingerprintPath(current.Files, path) {
						kind = "delete"
					}
					select {
					case output <- WatchEvent{BackendID: "watch:polling", Kind: kind, Path: path, Sequence: fmt.Sprintf("%d", sequence)}:
					case <-watchCtx.Done():
						return
					}
				}
				previous = current
			}
		}
	}()
	return output, nil
}

func (backend *LocalWatchBackend) Stop() error {
	if backend == nil {
		return nil
	}
	backend.mu.Lock()
	if backend.cancel != nil {
		backend.cancel()
		backend.cancel = nil
	}
	backend.started = false
	backend.mu.Unlock()
	return nil
}

func containsFingerprintPath(values []FileFingerprint, target string) bool {
	for _, value := range values {
		if value.Path == target {
			return true
		}
	}
	return false
}
