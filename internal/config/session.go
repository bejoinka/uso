package config

import (
	"bytes"
	"cmp"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrSessionExists is returned by CopySession when the target profile already
// has the session.
var ErrSessionExists = errors.New("session already exists")

// Session is one Claude Code session in a profile. Claude Code keeps it as a
// transcript, <profile>/projects/<project>/<id>.jsonl, next to a directory of
// the same name (subagent transcripts, large tool results, the /rename title),
// plus checkpoints for /rewind in <profile>/file-history/<id>.
type Session struct {
	Profile  string
	Project  string // the projects/ entry, named after the session's working directory
	ID       string
	Title    string    // set with /rename; empty if the session was never renamed
	Modified time.Time // when the transcript last changed
}

// ListSessions returns every session under Claude Code's accountsDir, or only
// those in profile from if it is set. A session that several profiles reach
// through a shared projects directory is returned once.
func ListSessions(accountsDir, from string) ([]Session, error) {
	profile := "*"
	if from != "" {
		profile = from
	}
	paths, err := filepath.Glob(filepath.Join(accountsDir, profile, "projects", "*", "*.jsonl"))
	if err != nil {
		return nil, err
	}

	var sessions []Session
	seen := map[string]bool{}
	for _, path := range paths {
		real, err := filepath.EvalSymlinks(path)
		if err != nil || seen[real] {
			continue // gone since the glob, or already listed
		}
		seen[real] = true
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		rel, err := filepath.Rel(accountsDir, path)
		if err != nil {
			return nil, err
		}
		parts := strings.Split(rel, string(filepath.Separator)) // profile, projects, project, <id>.jsonl
		s := Session{Profile: parts[0], Project: parts[2], ID: strings.TrimSuffix(parts[3], ".jsonl"), Modified: info.ModTime()}
		s.Title = readTitle(filepath.Join(filepath.Dir(path), s.ID, "custom-title.json"))
		sessions = append(sessions, s)
	}
	return sessions, nil
}

// FindSessions returns the sessions whose ID starts with ref or whose /rename
// title is ref, as ListSessions finds them.
func FindSessions(accountsDir, ref, from string) ([]Session, error) {
	if ref == "" {
		return nil, nil
	}
	all, err := ListSessions(accountsDir, from)
	if err != nil {
		return nil, err
	}
	var found []Session
	for _, s := range all {
		if strings.HasPrefix(s.ID, ref) || s.Title == ref {
			found = append(found, s)
		}
	}
	return found, nil
}

// Label returns a name to show for the session: its /rename title, else the
// title Claude Code generated for it, else its last prompt, else the start of
// its ID.
func (s Session) Label(accountsDir string) string {
	if s.Title != "" {
		return s.Title
	}
	var generated, prompt string
	// Claude Code appends these records again as a session goes on, so the
	// end of the transcript has the latest without reading all of it.
	for _, line := range tail(s.Transcript(accountsDir), 1<<20) {
		var r struct {
			AITitle    string `json:"aiTitle"`
			LastPrompt string `json:"lastPrompt"`
		}
		if json.Unmarshal(line, &r) != nil {
			continue
		}
		generated = cmp.Or(r.AITitle, generated)
		prompt = cmp.Or(r.LastPrompt, prompt)
	}
	return strings.Join(strings.Fields(cmp.Or(generated, prompt, s.ID[:min(8, len(s.ID))])), " ")
}

// tail returns the whole lines in the last n bytes of a file.
func tail(path string, n int64) [][]byte {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return nil
	}
	start := max(0, info.Size()-n)
	buf := make([]byte, info.Size()-start)
	if _, err := f.ReadAt(buf, start); err != nil && err != io.EOF {
		return nil
	}
	lines := bytes.Split(buf, []byte("\n"))
	if start > 0 {
		lines = lines[1:] // cut mid-record
	}
	return lines
}

func readTitle(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var t struct {
		CustomTitle string `json:"customTitle"`
	}
	json.Unmarshal(data, &t)
	return t.CustomTitle
}

// Transcript returns the path of the session's transcript.
func (s Session) Transcript(accountsDir string) string {
	return filepath.Join(accountsDir, s.Profile, "projects", s.Project, s.ID+".jsonl")
}

// WorkingDir returns the directory the session ran in, from the first
// transcript record that names one, or "" if none of the first records does.
func (s Session) WorkingDir(accountsDir string) string {
	f, err := os.Open(s.Transcript(accountsDir))
	if err != nil {
		return ""
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	for range 100 {
		var r struct {
			Cwd string `json:"cwd"`
		}
		if dec.Decode(&r) != nil {
			return ""
		}
		if r.Cwd != "" {
			return r.Cwd
		}
	}
	return ""
}

// CopySession copies a session into profile to, so that Claude Code can
// resume it there. It leaves the original in place and returns
// ErrSessionExists if to already has the session.
func CopySession(accountsDir string, s Session, to string) error {
	src, dst := filepath.Join(accountsDir, s.Profile), filepath.Join(accountsDir, to)
	transcript := filepath.Join("projects", s.Project, s.ID+".jsonl")
	if exists(filepath.Join(dst, transcript)) {
		return ErrSessionExists
	}

	for _, rel := range []string{
		filepath.Join("projects", s.Project, s.ID),
		filepath.Join("file-history", s.ID),
		// Last, so that a copy that fails partway leaves nothing to resume.
		transcript,
	} {
		if !exists(filepath.Join(src, rel)) {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dst, rel)), 0755); err != nil {
			return err
		}
		if err := copyDir(filepath.Join(src, rel), filepath.Join(dst, rel)); err != nil {
			return err
		}
	}
	return nil
}
