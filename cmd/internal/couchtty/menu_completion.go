package couchtty

import (
	"os"
	"sort"
	"strings"
)

const menuCompletionLimit = 200

type CompletionQuery struct {
	Directory      string
	EditablePrefix string
	NamePrefix     string
	IncludeHidden  bool
	Immediate      string
}

func SplitCompletionPath(path string) CompletionQuery {
	separator := string(os.PathSeparator)
	if path == "." || path == ".." {
		return CompletionQuery{Immediate: path + separator}
	}
	index := strings.LastIndex(path, separator)
	query := CompletionQuery{Directory: ".", NamePrefix: path}
	if index >= 0 {
		query.Directory = path[:index+1]
		query.EditablePrefix = query.Directory
		query.NamePrefix = path[index+1:]
	}
	query.IncludeHidden = strings.HasPrefix(query.NamePrefix, ".")
	return query
}

func (q CompletionQuery) CompletedPath(name string) string {
	return q.EditablePrefix + name + string(os.PathSeparator)
}

type CompletionEntry struct {
	Name      string
	Directory bool
}

type CompletionMatches struct {
	Paths     []string
	Truncated bool
}

type CompletionIdentity struct {
	FrameInstance uint64
	Generation    uint64
}

type CompletionRequest struct {
	Identity CompletionIdentity
	Path     string
}

type CompletionResult struct {
	Identity CompletionIdentity
	Matches  CompletionMatches
	Error    string
}

type CompletionAccumulator struct {
	query   CompletionQuery
	limit   int
	paths   []string
	matches int
}

func NewCompletionAccumulator(query CompletionQuery, limit int) CompletionAccumulator {
	if limit < 0 {
		limit = 0
	}
	return CompletionAccumulator{query: query, limit: limit}
}

func (a *CompletionAccumulator) Add(entries []CompletionEntry) {
	for _, entry := range entries {
		if !entry.Directory || !strings.HasPrefix(entry.Name, a.query.NamePrefix) ||
			(!a.query.IncludeHidden && strings.HasPrefix(entry.Name, ".")) {
			continue
		}
		a.matches++
		path := a.query.CompletedPath(entry.Name)
		index := sort.SearchStrings(a.paths, path)
		if len(a.paths) < a.limit {
			a.paths = append(a.paths, "")
			copy(a.paths[index+1:], a.paths[index:])
			a.paths[index] = path
			continue
		}
		if a.limit > 0 && index < a.limit {
			copy(a.paths[index+1:], a.paths[index:a.limit-1])
			a.paths[index] = path
		}
	}
}

func (a CompletionAccumulator) Result() CompletionMatches {
	return CompletionMatches{
		Paths:     append([]string(nil), a.paths...),
		Truncated: a.matches > a.limit,
	}
}
