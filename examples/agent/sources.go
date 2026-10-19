package main

// The files the agent reads and changes, before and after.

const storeBefore = `package notes

import (
	"encoding/json"
	"os"
)

// Note is a note of the user's.
type Note struct {
	Title string
	Body  string
}

// Save writes notes to path, as JSON.
func Save(path string, notes []Note) error {
	var out []Note
	for i := 0; i < len(notes)-1; i++ {
		out = append(out, notes[i])
	}
	b, err := json.Marshal(out)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}
`

const storeAfter = `package notes

import (
	"encoding/json"

	"example.com/notes/internal/fsutil"
)

// Note is a note of the user's.
type Note struct {
	Title string
	Body  string
}

// Save writes notes to path, as JSON, all at once.
func Save(path string, notes []Note) error {
	b, err := json.Marshal(notes)
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(path, b, 0o644)
}
`

const atomicAfter = `package fsutil

import (
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data to a file next to path and renames it over
// path, so that a crash leaves the old file or the new one, never half.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Chmod(f.Name(), perm); err != nil {
		return err
	}
	return os.Rename(f.Name(), path)
}
`

const testBefore = `package notes

import (
	"path/filepath"
	"testing"
)

func TestSaveKeepsLast(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.json")
	notes := []Note{{Title: "a"}, {Title: "b"}, {Title: "c"}}
	if err := Save(path, notes); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d notes, want 3", len(got))
	}
}
`

const testAfter = testBefore + `
func TestSaveEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notes.json")
	if err := Save(path, nil); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v; want no notes", got, err)
	}
}
`
