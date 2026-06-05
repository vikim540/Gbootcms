package parser

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type TemplateStore struct {
	mu       sync.RWMutex
	contents map[string]string
	mtimes   map[string]time.Time
	dir      string
	watcher  *fsnotify.Watcher
	parser   *TagParser
	stopCh   chan struct{}
}

func NewTemplateStore(dir string, parser *TagParser) (*TemplateStore, error) {
	ts := &TemplateStore{
		contents: make(map[string]string),
		mtimes:   make(map[string]time.Time),
		dir:      dir,
		parser:   parser,
		stopCh:   make(chan struct{}),
	}

	if err := ts.loadAll(); err != nil {
		return nil, err
	}

	if err := ts.startWatcher(); err != nil {
		return nil, err
	}

	return ts, nil
}

func (ts *TemplateStore) loadAll() error {
	return filepath.Walk(ts.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".html" || ext == ".htm" {
			rel, _ := filepath.Rel(ts.dir, path)
			rel = filepath.ToSlash(rel)
			data, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			ts.mu.Lock()
			ts.contents[rel] = string(data)
			ts.mtimes[rel] = info.ModTime()
			ts.mu.Unlock()
		}
		return nil
	})
}

func (ts *TemplateStore) startWatcher() error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	ts.watcher = w

	if err := w.Add(ts.dir); err != nil {
		return err
	}

	go ts.watchLoop()
	return nil
}

func (ts *TemplateStore) watchLoop() {
	for {
		select {
		case <-ts.stopCh:
			return
		case event, ok := <-ts.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write ||
				event.Op&fsnotify.Create == fsnotify.Create ||
				event.Op&fsnotify.Rename == fsnotify.Rename {
				ext := strings.ToLower(filepath.Ext(event.Name))
				if ext == ".html" || ext == ".htm" {
					rel, _ := filepath.Rel(ts.dir, event.Name)
					rel = filepath.ToSlash(rel)
					data, err := os.ReadFile(event.Name)
					if err == nil {
						ts.mu.Lock()
						ts.contents[rel] = string(data)
						ts.mtimes[rel] = time.Now()
						ts.mu.Unlock()
					}
				}
			}
		case <-ts.watcher.Errors:
		}
	}
}

func (ts *TemplateStore) Close() {
	close(ts.stopCh)
	if ts.watcher != nil {
		ts.watcher.Close()
	}
}

func (ts *TemplateStore) Get(name string) (string, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	content, ok := ts.contents[name]
	return content, ok
}

// Render template with include resolving
func (ts *TemplateStore) Render(name string) string {
	content, ok := ts.Get(name)
	if !ok {
		return "<!-- template not found: " + name + " -->"
	}

	visited := map[string]bool{name: true}
	content = ts.resolveIncludes(name, content, visited)

	// Only process non-include tags, include is already handled in resolveIncludes
	return ts.parser.RenderWithoutInclude(content)
}

func (ts *TemplateStore) resolveIncludes(currentPath string, content string, visited map[string]bool) string {
	re := ts.parser.re("include")
	if re == nil {
		return content
	}

	templatePrefix := ""
	if idx := strings.Index(currentPath, "/"); idx >= 0 {
		templatePrefix = currentPath[:idx+1]
	}

	maxDepth := 10
	for i := 0; i < maxDepth; i++ {
		found := false
		content = re.ReplaceAllStringFunc(content, func(match string) string {
			subs := re.FindStringSubmatch(match)
			if len(subs) < 2 {
				return match
			}
			incFile := subs[1]

			tryPaths := []string{incFile}
			if templatePrefix != "" && !strings.HasPrefix(incFile, templatePrefix) {
				tryPaths = append([]string{templatePrefix + incFile}, tryPaths...)
			}
			if !strings.HasSuffix(incFile, ".html") {
				tryPaths = append(tryPaths, incFile+".html")
				if templatePrefix != "" && !strings.HasPrefix(incFile, templatePrefix) {
					tryPaths = append(tryPaths, templatePrefix+incFile+".html")
				}
			}

			var incContent string
			var foundPath string
			for _, p := range tryPaths {
				if visited[p] {
					continue
				}
				if c, ok := ts.Get(p); ok {
					incContent = c
					foundPath = p
					break
				}
			}

			if foundPath == "" {
				return "<!-- include not found: " + incFile + " -->"
			}

			visited[foundPath] = true
			found = true
			incContent = ts.resolveIncludes(foundPath, incContent, visited)
			return incContent
		})
		if !found {
			break
		}
	}

	return content
}
