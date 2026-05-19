package ingest

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var (
	folderTextExt = map[string]bool{
		".md":   true, ".mdx": true, ".markdown": true,
		".txt":  true,
		".html": true, ".htm": true,
		".rst":  true,
		".org":  true,
	}
	folderIgnoreDirs = map[string]bool{
		"node_modules": true, ".git": true, ".next": true, "dist": true,
		"build": true, "out": true, ".turbo": true, ".vercel": true,
		"vendor": true, "__pycache__": true, ".venv": true, "venv": true,
	}
	folderMaxFiles      = 60
	folderMaxBytesEach  = 200 * 1024  // 200 KB per file
	folderMaxBytesTotal = 2 * 1024 * 1024 // 2 MB per ingest
)

type FolderDoc struct {
	Path    string
	Content string
}

// WalkFolder returns up to folderMaxFiles text-ish documents under root.
// Stops once total bytes exceed folderMaxBytesTotal.
func WalkFolder(root string, log func(string)) ([]FolderDoc, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}

	var docs []FolderDoc
	var totalBytes int
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // skip unreadable entries
		}
		name := d.Name()
		if d.IsDir() {
			if folderIgnoreDirs[name] || strings.HasPrefix(name, ".") && path != root {
				return fs.SkipDir
			}
			return nil
		}
		if len(docs) >= folderMaxFiles || totalBytes >= folderMaxBytesTotal {
			return filepath.SkipAll
		}
		if !folderTextExt[strings.ToLower(filepath.Ext(name))] {
			return nil
		}
		st, _ := d.Info()
		if st != nil && st.Size() > int64(folderMaxBytesEach) {
			log(fmt.Sprintf("skip %s — file too big (%d bytes)", path, st.Size()))
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		docs = append(docs, FolderDoc{Path: rel, Content: string(b)})
		totalBytes += len(b)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("no text/markdown files found under %s", root)
	}
	return docs, nil
}
