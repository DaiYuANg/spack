package watcher

import (
	"github.com/fsnotify/fsnotify"
	"go.uber.org/fx"
)

var Module = fx.Module("watcher", fx.Provide(newWatcher))

func newWatcher() (*fsnotify.Watcher, error) {
	return fsnotify.NewWatcher()
}
