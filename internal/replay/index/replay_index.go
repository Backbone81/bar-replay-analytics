package index

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/backbone81/bar-replay-analytics/internal/utils"
)

// ReplayIndex records which demo file of a directory holds which replay, so a tool fetching replays into the
// directory knows the replays it holds without reading every demo file, and without asking bar-db for their file
// names: a search returns no file names, only Client.Replay does.
type ReplayIndex struct {
	filePath string

	// fileNamesByID holds the file name of each replay by its ID.
	fileNamesByID map[string]string
}

// Open reads the index of dir, or starts an empty one if dir has no index file or does not exist. It lists
// the files of dir, but reads none of them.
func Open(filePath string) (*ReplayIndex, error) {
	data, err := os.ReadFile(filePath) //nolint:gosec // The caller must provide the file path.
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	if len(data) == 0 {
		data = []byte("{}")
	}

	fileNamesByID := make(map[string]string)
	if err := json.Unmarshal(data, &fileNamesByID); err != nil {
		return nil, err
	}

	return &ReplayIndex{
		filePath:      filePath,
		fileNamesByID: fileNamesByID,
	}, nil
}

func (i *ReplayIndex) Save() error {
	data, err := json.MarshalIndent(i.fileNamesByID, "", "  ")
	if err != nil {
		return err
	}

	//nolint:gosec // This file does not contain confidential data.
	if err := os.MkdirAll(filepath.Dir(i.filePath), 0o755); err != nil {
		return err
	}

	return utils.WriteFile(i.filePath, data, 0o644)
}

func (i *ReplayIndex) Add(replayId string, fileName string) {
	i.fileNamesByID[replayId] = fileName
}

func (i *ReplayIndex) Remove(replayId string) {
	delete(i.fileNamesByID, replayId)
}

func (i *ReplayIndex) Get(replayId string) (string, bool) {
	fileName, ok := i.fileNamesByID[replayId]
	return fileName, ok
}
