package util

import (
	"boogieman/src/model"
	"context"
	"github.com/fsnotify/fsnotify"
	"log"
)

type FileWatcher struct {
	watcher *fsnotify.Watcher
	logger  model.Logger
}

func Watcher(files []string, handler func(path, op string)) (fw *FileWatcher, err error) {
	fw = &FileWatcher{}
	fw.watcher, err = fsnotify.NewWatcher()
	fw.logger = model.NewChainLogger(model.DefaultLogger, "fileWatcher")
	if err != nil {
		return
	}

	fw.logger.Println("started")

	for _, file := range files {
		_ = fw.Add(file)
	}

	go func() {
		for {
			select {
			case event, ok := <-fw.watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write ||
					event.Op&fsnotify.Create == fsnotify.Create ||
					event.Op&fsnotify.Remove == fsnotify.Remove {
					fw.logger.Printf("file %s changed", event.Name)
					handler(event.Name, event.Op.String())
				}
			case err, ok := <-fw.watcher.Errors:
				if !ok {
					return
				}
				log.Println("error on file watching:", err)
			}
		}
	}()
	return
}

func (fw *FileWatcher) Add(file string) (err error) {
	err = fw.watcher.Add(file)
	if err != nil {
		fw.logger.Printf("error adding file %s to watching: %v\n", file, err)
	}
	fw.logger.Printf("file %s added to watching", file)
	return
}

func (fw *FileWatcher) Shutdown(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		fw.watcher.Close()
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-ctx.Done():
		fw.logger.Println("watcher finishing timeout")
	}
	return nil
}
