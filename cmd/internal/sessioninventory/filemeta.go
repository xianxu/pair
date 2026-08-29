package sessioninventory

import (
	"os"
	"time"
)

type fileMetadata struct {
	StableFileID    StableFileID
	GenerationToken GenerationToken
	MutationToken   MutationToken
	BirthTime       *time.Time
}

func readFileMetadata(path string, info os.FileInfo) (fileMetadata, error) {
	return platformFileMetadata(path, info)
}

func optionalTime(seconds, nanoseconds int64) *time.Time {
	if seconds == 0 && nanoseconds == 0 {
		return nil
	}
	value := time.Unix(seconds, nanoseconds).UTC()
	return &value
}
