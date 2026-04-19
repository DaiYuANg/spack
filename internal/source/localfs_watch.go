package source

import (
	"context"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/samber/oops"
)

func (s *localFS) Watch(ctx context.Context) (<-chan ChangeEvent, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, oops.Wrap(err)
	}
	if err := s.addWatchDirs(watcher); err != nil {
		_ = watcher.Close()
		return nil, err
	}

	changes := make(chan ChangeEvent, 1)
	go s.watchLoop(ctx, watcher, changes)
	return changes, nil
}

func (s *localFS) addWatchDirs(watcher *fsnotify.Watcher) error {
	if err := filepath.WalkDir(s.root, func(fullPath string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		return watcher.Add(fullPath)
	}); err != nil {
		return oops.Wrap(err)
	}
	return nil
}

func (s *localFS) watchLoop(ctx context.Context, watcher *fsnotify.Watcher, changes chan<- ChangeEvent) {
	defer close(changes)
	defer func() {
		if err := watcher.Close(); err != nil && s.logger != nil {
			s.logger.Debug("Close source watcher failed", slog.String("err", err.Error()))
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			s.handleWatchEvent(watcher, changes, event)
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			if err != nil && s.logger != nil {
				s.logger.Warn("Source watcher error", slog.String("err", err.Error()))
			}
		}
	}
}

func (s *localFS) handleWatchEvent(watcher *fsnotify.Watcher, changes chan<- ChangeEvent, event fsnotify.Event) {
	if event.Op.Has(fsnotify.Create) {
		s.addCreatedWatchDir(watcher, event.Name)
	}
	if !isContentWatchEvent(event) {
		return
	}

	change, ok := s.changeEvent(event)
	if !ok {
		return
	}
	select {
	case changes <- change:
	default:
	}
}

func (s *localFS) addCreatedWatchDir(watcher *fsnotify.Watcher, fullPath string) {
	info, err := os.Stat(fullPath)
	if err != nil || !info.IsDir() {
		return
	}
	if err := watcher.Add(fullPath); err != nil && s.logger != nil {
		s.logger.Warn("Add source watch directory failed",
			slog.String("path", fullPath),
			slog.String("err", err.Error()),
		)
	}
}

func (s *localFS) changeEvent(event fsnotify.Event) (ChangeEvent, bool) {
	rel, err := filepath.Rel(s.root, event.Name)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return ChangeEvent{}, false
	}
	return ChangeEvent{
		Path:     filepath.ToSlash(rel),
		FullPath: event.Name,
		Op:       event.Op.String(),
	}, true
}

func isContentWatchEvent(event fsnotify.Event) bool {
	return event.Op.Has(fsnotify.Create) ||
		event.Op.Has(fsnotify.Write) ||
		event.Op.Has(fsnotify.Remove) ||
		event.Op.Has(fsnotify.Rename)
}
