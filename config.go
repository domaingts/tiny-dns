package main

import (
	"encoding/json/v2"
	"errors"
	"log"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Config struct {
	WhiteList []string `json:"white_list"`
}

type FileProvider struct {
	config  atomic.Pointer[Config]
	watcher *fsnotify.Watcher
	done    chan struct{}
}

func NewFileProvider(path string) (*FileProvider, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	err = watcher.Add(filepath.Dir(path))
	if err != nil {
		watcher.Close()
		return nil, err
	}

	config, err := readConfig(path)
	if err != nil {
		watcher.Close()
		return nil, err
	}

	provider := &FileProvider{
		watcher: watcher,
		done:    make(chan struct{}),
	}

	provider.config.Store(config)

	go provider.watch(path)

	return provider, nil
}

func (f *FileProvider) updateConfig(config *Config) {
	f.config.Store(config)
}

func (f *FileProvider) watch(path string) {
	defer close(f.done)
	defer f.watcher.Close()

	const debounce = 200 * time.Millisecond

	timer := time.NewTimer(debounce)
	timer.Stop()
	defer timer.Stop()

	for {
		select {
		case event, ok := <-f.watcher.Events:
			if !ok {
				return
			}
			if event.Name != path {
				continue
			}
			if event.Has(fsnotify.Write | fsnotify.Create) {
				timer.Reset(debounce)
			}
		case <-timer.C:
			f.reload(path)
		case err, ok := <-f.watcher.Errors:
			if !ok {
				return
			}
			log.Println(err)

			if errors.Is(err, fsnotify.ErrEventOverflow) {
				timer.Reset(debounce)
			}
		}
	}
}

func (f *FileProvider) reload(path string) {
	config, err := readConfig(path)
	if err != nil {
		log.Println("reload config:", err)
		return
	}

	f.updateConfig(config)
}

func (f *FileProvider) InWhiteList(address string) bool {
	return slices.Contains(f.config.Load().WhiteList, address)
}

func (f *FileProvider) Close() error {
	err := f.watcher.Close()
	<-f.done
	return err
}

func readConfig(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(content, &config, json.RejectUnknownMembers(true))
	if err != nil {
		return nil, err
	}
	return &config, nil
}
