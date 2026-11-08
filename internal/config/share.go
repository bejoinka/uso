package config

import (
	"bytes"
	"os"
	"path/filepath"
)

// sharedItem is an entry of a tool's config directory that profiles can share.
type sharedItem struct {
	name string
	dir  bool
	// merge combines both profiles' contents when converting a profile that
	// has its own copy. Otherwise the source's copy is kept whole and the
	// profile's moves to the backup.
	merge bool
}

// sharedItems lists, per tool, the entries of its config directory that
// `uso add-profile --share` links to another profile. Everything else stays
// per profile, notably Claude Code's login and .claude.json (account,
// user-level MCP servers, trust), the account's server-managed settings, and
// runtime state.
var sharedItems = map[string][]sharedItem{
	"claude": {
		{name: "settings.json"},
		{name: "CLAUDE.md"},
		{name: "agents", dir: true, merge: true},
		{name: "commands", dir: true, merge: true},
		{name: "skills", dir: true, merge: true},
		{name: "output-styles", dir: true, merge: true},
		{name: "projects", dir: true, merge: true},
		// Installed plugins and marketplace git checkouts: mixing two
		// profiles' copies file by file would leave them inconsistent.
		{name: "plugins", dir: true},
	},
}

// ShareResult reports what ShareProfile did for one tool.
type ShareResult struct {
	Linked   []string // items the profile now shares
	BackedUp []string // the profile's own copies that conflicted, at their backup paths
}

// ShareProfile links each shared item of a tool's config directory in
// profile to the same item in source. If profile already has its own copy,
// its contents move into source's, keeping source's version wherever the two
// differ and moving profile's version into a uso-backup-* directory in the
// profile. Nothing is deleted except links and files identical to source's.
// It is safe to run again.
func ShareProfile(tool string, t Tool, profile, source string) (ShareResult, error) {
	var res ShareResult
	profileDir := filepath.Join(t.AccountsDir, profile)
	b := &backups{dir: profileDir}

	for _, item := range sharedItems[tool] {
		name := item.name
		dst := filepath.Join(profileDir, name)
		src := filepath.Join(t.AccountsDir, source, name)
		// If source shares this item from a third profile, link to that.
		if link, err := os.Readlink(src); err == nil {
			if !filepath.IsAbs(link) {
				link = filepath.Join(filepath.Dir(src), link)
			}
			src = link
		}

		if link, err := os.Readlink(dst); err == nil && link == src {
			res.Linked = append(res.Linked, name)
			continue
		}

		srcExists, dstExists := exists(src), exists(dst)
		switch {
		case !srcExists && !dstExists:
			if !item.dir {
				continue // nothing to share yet
			}
			// Create it so that what either profile adds later is shared.
			if err := os.MkdirAll(src, 0755); err != nil {
				return res, err
			}
		case !srcExists:
			if err := os.MkdirAll(filepath.Dir(src), 0755); err != nil {
				return res, err
			}
			if err := os.Rename(dst, src); err != nil {
				return res, err
			}
		case dstExists && item.merge:
			if err := merge(dst, src, name, b); err != nil {
				return res, err
			}
		case dstExists:
			var err error
			if sameFile(dst, src) || sameContent(dst, src) {
				err = os.Remove(dst)
			} else {
				err = b.move(dst, name)
			}
			if err != nil {
				return res, err
			}
		}

		if err := os.Symlink(src, dst); err != nil {
			return res, err
		}
		res.Linked = append(res.Linked, name)
	}

	res.BackedUp = b.paths
	return res, nil
}

// merge moves from into into, entry by entry. Where into already has a
// different file, from's version moves into the backup directory instead.
// from no longer exists afterwards.
func merge(from, into, rel string, b *backups) error {
	if !exists(into) {
		return os.Rename(from, into)
	}
	if sameFile(from, into) || sameContent(from, into) {
		return os.Remove(from)
	}
	if isRealDir(from) && isRealDir(into) {
		entries, err := os.ReadDir(from)
		if err != nil {
			return err
		}
		for _, e := range entries {
			name := e.Name()
			if err := merge(filepath.Join(from, name), filepath.Join(into, name), filepath.Join(rel, name), b); err != nil {
				return err
			}
		}
		return os.Remove(from)
	}
	return b.move(from, rel)
}

// backups moves conflicting files into one uso-backup-* directory inside a
// profile, created on first use.
type backups struct {
	dir    string // the profile directory
	backup string
	paths  []string
}

func (b *backups) move(from, rel string) error {
	if b.backup == "" {
		d, err := os.MkdirTemp(b.dir, "uso-backup-")
		if err != nil {
			return err
		}
		b.backup = d
	}
	to := filepath.Join(b.backup, rel)
	if err := os.MkdirAll(filepath.Dir(to), 0755); err != nil {
		return err
	}
	if err := os.Rename(from, to); err != nil {
		return err
	}
	b.paths = append(b.paths, to)
	return nil
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func isRealDir(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.IsDir()
}

// sameFile reports whether two paths lead to the same file, e.g. because
// one is a link to the other.
func sameFile(a, b string) bool {
	ai, errA := os.Stat(a)
	bi, errB := os.Stat(b)
	return errA == nil && errB == nil && os.SameFile(ai, bi)
}

// sameContent reports whether two paths are regular files with equal bytes,
// or links with the same target text (which then resolve alike, since uso
// only compares paths at the same depth).
func sameContent(a, b string) bool {
	if la, err := os.Readlink(a); err == nil {
		lb, err := os.Readlink(b)
		return err == nil && la == lb
	}
	ai, errA := os.Lstat(a)
	bi, errB := os.Lstat(b)
	if errA != nil || errB != nil || !ai.Mode().IsRegular() || !bi.Mode().IsRegular() || ai.Size() != bi.Size() {
		return false
	}
	da, errA := os.ReadFile(a)
	db, errB := os.ReadFile(b)
	return errA == nil && errB == nil && bytes.Equal(da, db)
}
