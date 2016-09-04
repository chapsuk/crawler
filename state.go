package crawler

import (
	"errors"
	"sync"
)

type Status int

type ItemType int

type Item struct {
	status Status
	itype  ItemType
}

type State struct {
	empty    bool
	progress map[string]Item
	mu       sync.Mutex
	wg       sync.WaitGroup
	storage  Storage
}

var (
	errHasMoreOrEqualStatus = errors.New("has more or equal status")
)

const (
	InFlightStatus Status = 1 + iota
	IgnoreStatus
	SavedStatus
)

const (
	PageType ItemType = 1 + iota
	AssetType
)

// NewState return new state instance
func NewState(s Storage) *State {
	return &State{
		progress: make(map[string]Item),
		storage:  s,
		empty:    true,
	}
}

// SetEmpty flag, if state is empty crawler run proccess from main url
func (s *State) SetEmpty(b bool) {
	s.empty = b
}

// IsEmpty return true if state is empty
func (s *State) IsEmpty() bool {
	return s.empty
}

// MarkAsInFlight set inFlight status and save it to storage
func (s *State) MarkAsInFlight(url string, t ItemType) error {
	s.wg.Add(1)
	if err := s.setStatus(url, t, InFlightStatus); err != nil {
		s.wg.Done()
		return err
	}
	return nil
}

// MarkAsSaved set saved status and save it to storage
func (s *State) MarkAsSaved(url string, t ItemType) error {
	defer s.wg.Done()
	return s.setStatus(url, t, SavedStatus)
}

// MarkAsIgnored set ignored status and save it to storage
func (s *State) MarkAsIgnored(url string, t ItemType) error {
	defer s.wg.Done()
	return s.setStatus(url, t, IgnoreStatus)
}

// set status and store it to storage
func (s *State) setStatus(url string, t ItemType, sts Status) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.progress[url]; ok {
		if v.status >= sts {
			return errHasMoreOrEqualStatus
		}
	}
	s.progress[url] = Item{status: sts, itype: t}
	if s.storage != nil {
		return s.storage.SetStatus(url, t, sts)
	}
	return nil
}

// IsSaved return true if page with url saved or mark as ignored
func (s *State) IsSaved(url string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.progress[url]; ok {
		return v.status == SavedStatus
	}
	return false
}

// GetInflight return inFlight items
func (s *State) GetInflight() map[string]Item {
	var res map[string]Item
	for k, v := range s.progress {
		if v.status == InFlightStatus {
			res[k] = v
		}
	}
	return res
}

// WaiteAll call waite work group
func (s *State) WaiteAll() {
	s.wg.Wait()
}
