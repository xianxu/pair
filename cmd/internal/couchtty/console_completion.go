package couchtty

import (
	"context"
	"errors"
	"io"
	"os"
)

const completionBatchSize = 128

type DirectoryBatchReader interface {
	ReadDirectoryBatches(context.Context, string, int, func([]CompletionEntry) bool) error
}

type directoryCursor interface {
	ReadDir(int) ([]os.DirEntry, error)
	Close() error
	Name() string
}

type OSDirectoryBatchReader struct {
	Open func(string) (directoryCursor, error)
}

func (reader OSDirectoryBatchReader) ReadDirectoryBatches(ctx context.Context, directory string, batchSize int, yield func([]CompletionEntry) bool) (resultErr error) {
	if directory == "" {
		directory = "."
	}
	open := reader.Open
	if open == nil {
		open = func(path string) (directoryCursor, error) { return os.Open(path) }
	}
	file, err := open(directory)
	if err != nil {
		return err
	}
	defer func() { resultErr = errors.Join(resultErr, file.Close()) }()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		entries, readErr := file.ReadDir(batchSize)
		batch := make([]CompletionEntry, 0, len(entries))
		for _, entry := range entries {
			directory := entry.IsDir()
			if entry.Type()&os.ModeSymlink != 0 {
				info, statErr := os.Stat(file.Name() + string(os.PathSeparator) + entry.Name())
				directory = statErr == nil && info.IsDir()
			}
			if directory {
				batch = append(batch, CompletionEntry{Name: entry.Name(), Directory: true})
			}
		}
		if len(batch) > 0 && !yield(batch) {
			return context.Canceled
		}
		if errors.Is(readErr, io.EOF) {
			return nil
		}
		if readErr != nil {
			return readErr
		}
	}
}

type menuCompletionResult struct {
	result CompletionResult
}

func (c *Console) SetDirectoryBatchReader(reader DirectoryBatchReader) {
	c.mu.Lock()
	c.directoryReader = reader
	c.mu.Unlock()
}

func (c *Console) advanceMenuCompletion(event latestScheduleEvent[CompletionRequest, CompletionIdentity]) {
	c.mu.Lock()
	var effects []latestScheduleEffect[CompletionRequest, CompletionIdentity]
	c.completionSchedule, effects = advanceLatestSchedule(c.completionSchedule, event,
		func(request CompletionRequest) CompletionIdentity { return request.Identity },
		func(identity CompletionIdentity) bool { return identity.FrameInstance != 0 && identity.Generation != 0 },
	)
	c.mu.Unlock()
	for _, effect := range effects {
		switch effect.Kind {
		case latestCancel:
			c.mu.Lock()
			cancel := c.completionCancel
			matches := c.completionRunning == effect.Identity
			c.mu.Unlock()
			if matches && cancel != nil {
				cancel()
			}
		case latestStart:
			c.startMenuCompletion(effect.Request)
		}
	}
}

func (c *Console) startMenuCompletion(request CompletionRequest) {
	ctx, cancel := context.WithCancel(c.lifetime)
	c.mu.Lock()
	c.completionCancel = cancel
	c.completionRunning = request.Identity
	reader := c.directoryReader
	c.mu.Unlock()
	c.workers.Add(1)
	go func() {
		defer c.workers.Done()
		query := SplitCompletionPath(request.Path)
		accumulator := NewCompletionAccumulator(query, menuCompletionLimit)
		err := reader.ReadDirectoryBatches(ctx, query.Directory, completionBatchSize, func(batch []CompletionEntry) bool {
			accumulator.Add(batch)
			return ctx.Err() == nil
		})
		if errors.Is(err, context.Canceled) {
			err = nil
		}
		result := CompletionResult{Identity: request.Identity, Matches: accumulator.Result()}
		if err != nil {
			result.Error = err.Error()
		}
		select {
		case c.completionResults <- menuCompletionResult{result: result}:
		case <-c.stop:
		}
	}()
}

func (c *Console) finishMenuCompletion(completed menuCompletionResult) {
	identity := completed.result.Identity
	c.mu.Lock()
	if c.completionRunning == identity {
		if c.completionCancel != nil {
			c.completionCancel()
		}
		c.completionCancel = nil
		c.completionRunning = CompletionIdentity{}
	}
	var effects []MenuEffect
	if c.menuReady {
		c.menu, effects = ReduceMenu(c.menu, MenuEvent{Kind: MenuEventCompletionResult, Completion: &completed.result})
	}
	panelFocused := c.focus.IsPanel()
	c.mu.Unlock()
	c.advanceMenuCompletion(latestScheduleEvent[CompletionRequest, CompletionIdentity]{Kind: latestFinished, Identity: identity})
	c.dispatchMenuEffects(effects)
	if panelFocused {
		c.showMenu()
	}
}
