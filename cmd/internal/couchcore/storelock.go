package couchcore

import "os"

type threadStoreLock struct{ file *os.File }

func (l *threadStoreLock) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	file := l.file
	l.file = nil
	return unlockThreadStoreFile(file)
}
