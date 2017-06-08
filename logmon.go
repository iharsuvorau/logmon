// Package logmon implements file watching functionality like a tail -f program.
package logmon

import (
	"errors"
	"os"
	"sync"
	"time"
)

// delay between file reads
var sleeptime = 1 * time.Second

// Watcher interface represents an object which can watch itself for new data comming in.
type Watcher interface {
	Watch()
}

// File abstracts the text file data.
type File struct {
	ID           uint64
	Path         string
	Position     int64
	positionLock sync.Mutex
	Data         chan []byte
	Errs         chan error
}

// NewFile return a file object with channels for communication.
func NewFile(path string) *File {
	return &File{
		Path:     path,
		Position: 0,
		Data:     make(chan []byte),
		Errs:     make(chan error),
	}
}

// Watch tracks file changes and report new data or errors to the channels.
func (f *File) Watch() {
	var file *os.File
	var err error
	var stat os.FileInfo
	var newpos int64

	for {
		if file, err = os.Open(f.Path); err != nil {
			err = errors.New("failed to open the file: " + err.Error())
			f.Errs <- err
			return
		}

		if stat, err = file.Stat(); err != nil {
			err = errors.New("failed to stat the file: " + err.Error())
			f.Errs <- err
			return
		}

		newpos = stat.Size()
		if newpos > f.Position {
			f.positionLock.Lock()
			size := newpos - f.Position
			if size < 0 {
				size = 0
			}
			b := make([]byte, size)
			if _, err = file.ReadAt(b, f.Position); err != nil {
				err = errors.New("failed to read: " + err.Error())
				f.Errs <- err
				return
			}
			f.Data <- b
			f.Position = newpos
			f.positionLock.Unlock()
		}

		file.Close()
		time.Sleep(sleeptime)
	}
}

// WatchList accumulates all files to watch.
type WatchList struct {
	m           map[uint64]*File // local storage
	counter     uint64           // list scoped ID counter
	counterLock sync.Mutex
}

// NewWatchList creates a new WatchList and returns a pointer to it.
func NewWatchList() *WatchList {
	return &WatchList{m: make(map[uint64]*File)}
}

// Add adds a new file to the watch list if it's not already there.
func (w *WatchList) Add(path string) (uint64, error) {
	if w.GetByPath(path) != nil {
		return 0, errors.New("the file already exist")
	}
	f := NewFile(path)
	w.counterLock.Lock()
	w.counter++
	f.ID = w.counter
	w.counterLock.Unlock()
	w.m[f.ID] = f
	return f.ID, nil
}

// Get returns a file's pointer by the ID.
func (w *WatchList) Get(id uint64) *File {
	return w.m[id]
}

// GetByPath returns a file's pointer by the filepath.
func (w *WatchList) GetByPath(path string) *File {
	for _, v := range w.m {
		if v.Path == path {
			return v
		}
	}
	return nil
}

// Del deletes a file object from the watch list.
func (w *WatchList) Del(id uint64) {
	delete(w.m, id)
}

// ListPathes return a map of files in the watch list.
func (w *WatchList) ListPathes() map[uint64]string {
	list := make(map[uint64]string)
	for k, v := range w.m {
		list[k] = v.Path
	}
	return list
}
