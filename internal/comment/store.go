package comment

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ErrNotFound is returned when a comment ID does not exist in the store.
var ErrNotFound = errors.New("comment not found")

// Comment represents a single review comment on a diff line.
type Comment struct {
	ID             string    `yaml:"id"`
	Line           int       `yaml:"line"`
	EndLine        int       `yaml:"end_line,omitempty"`
	ContentSnippet string    `yaml:"content_snippet"`
	Body           string    `yaml:"body"`
	CreatedAt      time.Time `yaml:"created_at"`
}

// ReviewState is the on-disk YAML format for a file's comments, matching crit's schema.
type ReviewState struct {
	File     string    `yaml:"file"`
	Comments []Comment `yaml:"comments"`
}

// Store holds in-memory comments keyed by file path, backed by .crit/ YAML on disk.
type Store struct {
	repoRoot string
	comments map[string][]Comment // keyed by file path
}

// NewStore creates a new comment store rooted at the given repo directory.
func NewStore(repoRoot string) *Store {
	return &Store{
		repoRoot: repoRoot,
		comments: make(map[string][]Comment),
	}
}

// Comments returns the comments for a file. Returns an empty slice if none exist.
func (s *Store) Comments(file string) []Comment {
	if c, ok := s.comments[file]; ok {
		return c
	}
	return []Comment{}
}

// HasComment returns true if a comment already exists on the given line for the file.
func (s *Store) HasComment(file string, line int) bool {
	for _, c := range s.comments[file] {
		if c.Line == line {
			return true
		}
	}
	return false
}

// AddComment creates a new comment and persists it to disk.
func (s *Store) AddComment(file string, line, endLine int, snippet, body string) error {
	c := Comment{
		ID:             uuid.NewString()[:8],
		Line:           line,
		EndLine:        endLine,
		ContentSnippet: snippet,
		Body:           body,
		CreatedAt:      time.Now().UTC(),
	}
	s.comments[file] = append(s.comments[file], c)
	return s.persist(file)
}

// EditComment updates the body of a comment by ID and persists the change.
func (s *Store) EditComment(id, newBody string) error {
	for file, comments := range s.comments {
		for i, c := range comments {
			if c.ID == id {
				s.comments[file][i].Body = newBody
				return s.persist(file)
			}
		}
	}
	return fmt.Errorf("%w: %s", ErrNotFound, id)
}

// DeleteComment removes a comment by ID and persists the change.
func (s *Store) DeleteComment(id string) error {
	for file, comments := range s.comments {
		for i, c := range comments {
			if c.ID == id {
				s.comments[file] = append(comments[:i], comments[i+1:]...)
				return s.persist(file)
			}
		}
	}
	return fmt.Errorf("%w: %s", ErrNotFound, id)
}

// LoadComments reads comments for a file from the .crit/ YAML on disk.
func (s *Store) LoadComments(file string) error {
	state, err := readReviewFile(reviewPath(s.repoRoot, file))
	if err != nil {
		return fmt.Errorf("load comments for %s: %w", file, err)
	}
	if state != nil {
		s.comments[file] = state.Comments
	}
	return nil
}

// LoadAll loads comments for all given file paths from disk.
func (s *Store) LoadAll(files []string) error {
	for _, f := range files {
		if err := s.LoadComments(f); err != nil {
			return fmt.Errorf("load comments for %s: %w", f, err)
		}
	}
	return nil
}

// persist writes the current comments for a file to .crit/ YAML.
func (s *Store) persist(file string) error {
	if err := ensureCritDir(s.repoRoot); err != nil {
		return fmt.Errorf("persist %s: %w", file, err)
	}
	state := ReviewState{
		File:     file,
		Comments: s.comments[file],
	}
	return writeReviewFile(reviewPath(s.repoRoot, file), &state)
}
