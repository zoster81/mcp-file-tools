package httptransport

import (
	"sync"
	"time"
)

type sessionGate struct {
	mu       sync.Mutex
	maximum  int
	timeout  time.Duration
	pending  int
	closed   bool
	sessions map[string]*sessionEntry
}

type sessionEntry struct {
	generation uint64
	active     int
	timer      *time.Timer
}

type sessionReservation struct {
	gate *sessionGate
	once sync.Once
}

func newSessionGate(maximum int, timeout time.Duration) *sessionGate {
	return &sessionGate{
		maximum:  maximum,
		timeout:  timeout,
		sessions: make(map[string]*sessionEntry),
	}
}

func (gate *sessionGate) reserve() (*sessionReservation, bool) {
	gate.mu.Lock()
	defer gate.mu.Unlock()
	if gate.closed || len(gate.sessions)+gate.pending >= gate.maximum {
		return nil, false
	}
	gate.pending++
	return &sessionReservation{gate: gate}, true
}

func (reservation *sessionReservation) commit(sessionID string) {
	if reservation == nil || reservation.gate == nil {
		return
	}
	reservation.once.Do(func() { reservation.gate.commitReservation(sessionID) })
}

func (reservation *sessionReservation) cancel() {
	if reservation == nil || reservation.gate == nil {
		return
	}
	reservation.once.Do(reservation.gate.cancelReservation)
}

func (gate *sessionGate) commitReservation(sessionID string) {
	gate.mu.Lock()
	defer gate.mu.Unlock()
	if gate.pending > 0 {
		gate.pending--
	}
	if gate.closed || sessionID == "" {
		return
	}
	if existing := gate.sessions[sessionID]; existing != nil {
		gate.stopEntryLocked(existing)
	}
	entry := &sessionEntry{generation: 1}
	gate.sessions[sessionID] = entry
	gate.scheduleLocked(sessionID, entry)
}

func (gate *sessionGate) cancelReservation() {
	gate.mu.Lock()
	defer gate.mu.Unlock()
	if gate.pending > 0 {
		gate.pending--
	}
}

func (gate *sessionGate) contains(sessionID string) bool {
	gate.mu.Lock()
	defer gate.mu.Unlock()
	return !gate.closed && gate.sessions[sessionID] != nil
}

func (gate *sessionGate) acquire(sessionID string) (func(), bool) {
	gate.mu.Lock()
	entry := gate.sessions[sessionID]
	if gate.closed || entry == nil {
		gate.mu.Unlock()
		return nil, false
	}
	gate.stopEntryLocked(entry)
	entry.generation++
	entry.active++
	gate.mu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() { gate.releaseActive(sessionID, entry) })
	}, true
}

func (gate *sessionGate) release(sessionID string) {
	gate.mu.Lock()
	defer gate.mu.Unlock()
	entry := gate.sessions[sessionID]
	if entry == nil {
		return
	}
	gate.stopEntryLocked(entry)
	delete(gate.sessions, sessionID)
}

func (gate *sessionGate) releaseActive(sessionID string, acquired *sessionEntry) {
	gate.mu.Lock()
	defer gate.mu.Unlock()
	entry := gate.sessions[sessionID]
	if entry == nil || entry != acquired || entry.active == 0 {
		return
	}
	entry.active--
	if entry.active == 0 {
		entry.generation++
		gate.scheduleLocked(sessionID, entry)
	}
}

func (gate *sessionGate) closeAll() {
	gate.mu.Lock()
	defer gate.mu.Unlock()
	if gate.closed {
		return
	}
	gate.closed = true
	gate.pending = 0
	for sessionID, entry := range gate.sessions {
		gate.stopEntryLocked(entry)
		delete(gate.sessions, sessionID)
	}
}

func (gate *sessionGate) scheduleLocked(sessionID string, entry *sessionEntry) {
	generation := entry.generation
	entry.timer = time.AfterFunc(gate.timeout, func() {
		gate.expire(sessionID, generation)
	})
}

func (gate *sessionGate) expire(sessionID string, generation uint64) {
	gate.mu.Lock()
	defer gate.mu.Unlock()
	entry := gate.sessions[sessionID]
	if entry == nil || entry.generation != generation || entry.active != 0 {
		return
	}
	delete(gate.sessions, sessionID)
}

func (gate *sessionGate) stopEntryLocked(entry *sessionEntry) {
	if entry.timer != nil {
		entry.timer.Stop()
		entry.timer = nil
	}
}
