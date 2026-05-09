package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	servePort = ":8000"
	serveDir  = outDir
)

var watchedExts = map[string]bool{
	".md": true, ".html": true, ".css": true, ".js": true,
	".yaml": true, ".yml": true, ".json": true,
	".png": true, ".ico": true, ".svg": true, ".webmanifest": true,
}

func serve() error {
	if err := build(); err != nil {
		return err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	if err := addWatchDirs(watcher, "."); err != nil {
		return err
	}

	go runWatcher(watcher)

	fs := http.FileServer(http.Dir(serveDir))
	http.Handle("/", fs)
	fmt.Printf("Serving %s on http://localhost%s\n", serveDir, servePort)
	fmt.Println("Watching for file changes... (Press Ctrl+C to stop)")
	return http.ListenAndServe(servePort, nil)
}

func addWatchDirs(w *fsnotify.Watcher, root string) error {
	return filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() {
			return nil
		}
		base := filepath.Base(p)
		if base == ".git" || base == ".direnv" || base == ".venv" || base == outDir {
			return filepath.SkipDir
		}
		return w.Add(p)
	})
}

func runWatcher(w *fsnotify.Watcher) {
	var (
		mu    sync.Mutex
		timer *time.Timer
	)
	trigger := func() {
		mu.Lock()
		defer mu.Unlock()
		if timer != nil {
			timer.Stop()
		}
		timer = time.AfterFunc(150*time.Millisecond, func() {
			fmt.Println("Regenerating content...")
			if err := build(); err != nil {
				fmt.Fprintf(os.Stderr, "build error: %v\n", err)
				return
			}
			fmt.Println("Content regenerated")
		})
	}
	for {
		select {
		case ev, ok := <-w.Events:
			if !ok {
				return
			}
			if shouldRebuild(ev) {
				fmt.Printf("File %s %s\n", ev.Name, ev.Op)
				if ev.Op&fsnotify.Create != 0 {
					if info, err := os.Stat(ev.Name); err == nil && info.IsDir() {
						_ = w.Add(ev.Name)
					}
				}
				trigger()
			}
		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			fmt.Fprintf(os.Stderr, "watch error: %v\n", err)
		}
	}
}

func shouldRebuild(ev fsnotify.Event) bool {
	if ev.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) == 0 {
		return false
	}
	p := filepath.ToSlash(ev.Name)
	if strings.Contains(p, "/"+outDir+"/") || strings.HasPrefix(p, outDir+"/") {
		return false
	}
	if strings.Contains(p, "/.git/") || strings.Contains(p, "/.direnv/") {
		return false
	}
	ext := strings.ToLower(filepath.Ext(ev.Name))
	if ext == "" {
		return false
	}
	return watchedExts[ext]
}
