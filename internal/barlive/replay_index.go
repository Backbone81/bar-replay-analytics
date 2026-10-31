package barlive

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/backbone81/bar-replay-analytics-go/internal/demo"
)

// ReplayIndexFileName is the name of the file in which a ReplayIndex keeps its entries, in the directory it indexes.
const ReplayIndexFileName = "index.json"

// ReplayIndex records which demo file of a directory holds which replay, so a tool fetching replays into the
// directory knows the replays it holds without reading every demo file, and without asking bar-db for their file
// names: a search returns no file names, only Client.Replay does.
//
// The index heals itself if files are added, renamed or removed by hand: an entry counts only while its file exists,
// and Fetch looks up a replay which has no such entry, takes a file of its name into the index if that file holds its
// game, and downloads it otherwise. Save drops the entries whose files are gone.
//
// A ReplayIndex is safe for concurrent use.
type ReplayIndex struct {
	dir string

	mutex sync.Mutex

	// fileNamesByID holds the file name of each replay by its ID.
	fileNamesByID map[string]string

	// filesInDir holds the names of the filesInDir in dir.
	filesInDir map[string]bool
}

// OpenReplayIndex reads the index of dir, or starts an empty one if dir has no index file or does not exist. It lists
// the files of dir, but reads none of them.
func OpenReplayIndex(dir string) (*ReplayIndex, error) {
	index := &ReplayIndex{
		dir:           dir,
		fileNamesByID: make(map[string]string),
		filesInDir:    make(map[string]bool),
	}

	entries, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	for _, entry := range entries {
		index.filesInDir[entry.Name()] = true
	}

	data, err := os.ReadFile(index.path())
	if errors.Is(err, fs.ErrNotExist) {
		return index, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &index.fileNamesByID); err != nil {
		return nil, fmt.Errorf("%s: %w (remove the file to rebuild the index)", index.path(), err)
	}
	return index, nil
}

// path returns the path of the index file.
func (i *ReplayIndex) path() string {
	return filepath.Join(i.dir, ReplayIndexFileName)
}

// Has reports whether the index has an entry for the replay with ID id whose file exists.
func (i *ReplayIndex) Has(id string) bool {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	fileName, found := i.fileNamesByID[id]
	return found && i.filesInDir[fileName]
}

// Len returns the number of entries whose files exist.
func (i *ReplayIndex) Len() int {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	count := 0
	for _, fileName := range i.fileNamesByID {
		if i.filesInDir[fileName] {
			count++
		}
	}
	return count
}

// FetchResult is what Fetch did for a replay.
type FetchResult struct {
	// FileName is the name of the replay's demo file in the directory.
	FileName string

	// Downloaded is true if Fetch downloaded the demo file, and false if the directory already held it.
	Downloaded bool
}

// Fetch makes the directory hold the replay with ID id and the index record it. If Has(id), it does nothing.
// Otherwise it asks bar-db for the replay's file name: if the directory holds a file of that name, Fetch reads the
// file's header and records it if it holds the replay's game, else it reports an error matching fs.ErrExist, as
// another game's replay has the same name. If there is no such file, Fetch downloads it with DownloadDemoIntoDir.
func (i *ReplayIndex) Fetch(ctx context.Context, client *Client, id string) (FetchResult, error) {
	i.mutex.Lock()
	fileName, found := i.fileNamesByID[id]
	present := found && i.filesInDir[fileName]
	i.mutex.Unlock()

	if present {
		return FetchResult{FileName: fileName}, nil
	}

	replay, err := client.Replay(ctx, id)
	if err != nil {
		return FetchResult{}, err
	}

	if err := ValidateDemoFileName(replay.FileName); err != nil {
		return FetchResult{}, err
	}

	result := FetchResult{FileName: replay.FileName}
	_, err = os.Lstat(filepath.Join(i.dir, replay.FileName))
	switch {
	case err == nil:
		if err := i.validateExistingFileHoldsReplay(replay); err != nil {
			return FetchResult{}, err
		}
	case errors.Is(err, fs.ErrNotExist):
		if _, err := client.DownloadDemoIntoDir(ctx, replay, i.dir); err != nil {
			return FetchResult{}, err
		}
		result.Downloaded = true
	default:
		return FetchResult{}, err
	}

	i.mutex.Lock()
	i.fileNamesByID[replay.ID] = replay.FileName
	i.filesInDir[replay.FileName] = true
	i.mutex.Unlock()
	return result, nil
}

// validateExistingFileHoldsReplay reads the header of the file named Replay.FileName, which the directory already
// holds, and returns nil if it holds the game of replay (Replay.ID). If it holds another game, a different replay has
// the same file name, and it returns an error matching fs.ErrExist.
func (i *ReplayIndex) validateExistingFileHoldsReplay(replay *Replay) error {
	filePath := filepath.Join(i.dir, replay.FileName)
	header, err := demo.ReadDemoFileHeaderFromFilePath(filePath)
	if err != nil {
		return fmt.Errorf("%s: %w", filePath, err)
	}

	if gameID := header.GameIDAsString(); gameID != replay.ID {
		return fmt.Errorf("%s holds game %s, not %s: %w", filePath, gameID, replay.ID, fs.ErrExist)
	}
	return nil
}

// Save writes the entries whose files exist to the index file, which it replaces only once the new one is complete.
// It creates the directory if it is missing.
func (i *ReplayIndex) Save() error {
	i.mutex.Lock()
	entries := make(map[string]string, len(i.fileNamesByID))
	for id, fileName := range i.fileNamesByID {
		if i.filesInDir[fileName] {
			entries[id] = fileName
		}
	}
	i.mutex.Unlock()

	// json.Marshal sorts the keys, so the file changes only where entries change.
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(i.dir, dirPermissions); err != nil {
		return err
	}
	return i.writeFileAtomically(i.path(), append(data, '\n'))
}

// writeFileAtomically writes data to filePath with ".part" appended and renames that file to filePath, so filePath
// holds either its old or its new content.
func (i *ReplayIndex) writeFileAtomically(filePath string, data []byte) error {
	partPath := filePath + ".part"
	if err := os.WriteFile(partPath, data, filePermissions); err != nil {
		return errors.Join(err, os.Remove(partPath))
	}
	if err := os.Rename(partPath, filePath); err != nil {
		return errors.Join(err, os.Remove(partPath))
	}
	return nil
}
