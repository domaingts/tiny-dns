package main

import (
	"encoding/json/v2"
	"log"
	"os"
	"slices"
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
)

type Config struct {
	WhiteList []string `json:"white_list"`
}

type FileProvider struct {
	config atomic.Pointer[Config]
}

func NewFileProvider(path string) (*FileProvider, error) {

	provider := &FileProvider{}

	config, err := readConfig(path)
	if err != nil {
		return nil, err
	}
	provider.config.Store(config)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	err = watcher.Add(path)
	if err != nil {
		watcher.Close()
		return nil, err
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) {
					config, err := readConfig(path)
					if err != nil {
						log.Println(err)
						continue
					}
					provider.updateConfig(config)
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println(err)
			}
		}
	}()
	return provider, nil
}

func (f *FileProvider) updateConfig(config *Config) {
	f.config.Store(config)
}

func (f *FileProvider) InWhiteList(address string) bool {
	return slices.Contains(f.config.Load().WhiteList, address)
}

func readConfig(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(content, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
