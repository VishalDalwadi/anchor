// Package store loads and saves anchor's JSON record arrays through a
// storage.Backend.
package store

import (
	"encoding/json"
	"fmt"

	"github.com/VishalDalwadi/anchor/internal/model"
	"github.com/VishalDalwadi/anchor/internal/storage"
)

const (
	tasksPath       = "tasks.json"
	goalsPath       = "goals.json"
	watchPath       = "watchlist.json"
	recurringPath   = "recurring.json"
	collectionsPath = "collections.json"
)

type Store struct {
	backend storage.Backend
}

func New(backend storage.Backend) *Store {
	return &Store{backend: backend}
}

func loadJSON[T any](s *Store, path string) ([]T, error) {
	data, err := s.backend.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	if data == nil {
		return []T{}, nil
	}
	var records []T
	if err := json.Unmarshal(data, &records); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return records, nil
}

func saveJSON[T any](s *Store, path string, records []T) error {
	data, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding %s: %w", path, err)
	}
	if err := s.backend.WriteFile(path, data); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

func (s *Store) LoadTasks() ([]model.Task, error) { return loadJSON[model.Task](s, tasksPath) }
func (s *Store) SaveTasks(t []model.Task) error   { return saveJSON(s, tasksPath, t) }

func (s *Store) LoadGoals() ([]model.Goal, error) { return loadJSON[model.Goal](s, goalsPath) }
func (s *Store) SaveGoals(g []model.Goal) error   { return saveJSON(s, goalsPath, g) }

func (s *Store) LoadWatchItems() ([]model.WatchItem, error) {
	return loadJSON[model.WatchItem](s, watchPath)
}
func (s *Store) SaveWatchItems(w []model.WatchItem) error { return saveJSON(s, watchPath, w) }

func (s *Store) LoadRecurring() ([]model.RecurringTemplate, error) {
	return loadJSON[model.RecurringTemplate](s, recurringPath)
}
func (s *Store) SaveRecurring(r []model.RecurringTemplate) error {
	return saveJSON(s, recurringPath, r)
}

func (s *Store) LoadCollections() ([]model.Collection, error) {
	return loadJSON[model.Collection](s, collectionsPath)
}
func (s *Store) SaveCollections(c []model.Collection) error {
	return saveJSON(s, collectionsPath, c)
}

// WriteLog writes a generated daily briefing. log/ is output-only and is
// never read back in as a data source.
func (s *Store) WriteLog(dateYYYYMMDD, content string) error {
	return s.backend.WriteFile(fmt.Sprintf("log/%s.md", dateYYYYMMDD), []byte(content))
}
