package agentfixture

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/xianxu/pair/cmd/internal/resumeform"
	"github.com/xianxu/pair/cmd/internal/sessioninventory"
)

var nativeUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

type conversation struct {
	*os.File
	id, path, cwd, agent string
	turn                 uint64
}

func openConversation(c Config) (*conversation, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	id, resume := resumeform.Extract(c.Agent, c.Args), false
	if id != "" {
		resume = true
	}
	if c.Agent == "codex" {
		for i, a := range c.Args {
			if a == "--" {
				break
			}
			if a == "resume" && i+1 < len(c.Args) {
				id = c.Args[i+1]
				resume = true
				break
			}
		}
	}
	if !resume {
		for i, a := range c.Args {
			if a == "--" {
				break
			}
			if a == "--session-id" && i+1 < len(c.Args) {
				id = c.Args[i+1]
			}
			if v, ok := strings.CutPrefix(a, "--session-id="); ok {
				id = v
			}
		}
	}
	if id == "" {
		sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", c.RunID, c.ControlDir, os.Getpid())))
		sum[6] = (sum[6] & 15) | 64
		sum[8] = (sum[8] & 63) | 128
		h := hex.EncodeToString(sum[:16])
		id = h[:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
	}
	if !nativeUUID.MatchString(id) {
		return nil, errors.New("fixture native ID must be a full UUID")
	}
	runtime := sessioninventory.NewOSRuntime(c.Home, "")
	if c.Agent == "codex" {
		runtime, err = runtime.WithCodexHome(c.CodexHome)
		if err != nil {
			return nil, err
		}
	}
	roots := runtime.NativeRoots(sessioninventory.Agent(c.Agent))
	if len(roots) != 1 {
		return nil, errors.New("fixture agent requires one canonical native root")
	}
	root := roots[0].Path
	path := filepath.Join(root, strings.ReplaceAll(cwd, "/", "-"), id+".jsonl")
	if c.Agent == "codex" {
		path = filepath.Join(root, time.Now().UTC().Format("2006/01/02"), "rollout-fixture-"+id+".jsonl")
	}
	if resume {
		// Dates/project directories may differ from today's launch. Inspect only
		// the bounded private fixture corpus, then use production native validation.
		found := ""
		visited := 0
		err = filepath.WalkDir(root, func(p string, d os.DirEntry, e error) error {
			if e != nil {
				return e
			}
			visited++
			if visited > 1024 {
				return errors.New("fixture native corpus limit")
			}
			if d.Type()&os.ModeSymlink != 0 {
				return errors.New("symlink in fixture native corpus")
			}
			if !d.IsDir() && strings.HasSuffix(d.Name(), id+".jsonl") {
				if found != "" {
					return errors.New("ambiguous fixture native ID")
				}
				found = p
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
		if found == "" {
			return nil, errors.New("fixture native resume transcript missing")
		}
		path = found
		if !sessioninventory.NativeSessionCandidateExists(runtime, sessioninventory.Agent(c.Agent), id) {
			return nil, errors.New("fixture native transcript validation refused resume")
		}
	} else {
		if err := privateDirectory(filepath.Dir(path)); err != nil {
			return nil, err
		}
	}
	flags := os.O_WRONLY | os.O_APPEND
	if !resume {
		flags |= os.O_CREATE | os.O_EXCL
	}
	f, err := os.OpenFile(path, flags, 0600)
	if err != nil {
		return nil, err
	}
	cv := &conversation{File: f, id: id, path: path, cwd: cwd, agent: c.Agent}
	if !resume && c.Agent == "codex" {
		err = cv.record(map[string]any{"timestamp": time.Now().UTC().Format(time.RFC3339Nano), "type": "session_meta", "payload": map[string]any{"id": id, "source": "cli", "cwd": cwd, "parent_thread_id": nil}})
	}
	if err != nil {
		f.Close()
		return nil, err
	}
	return cv, nil
}
func (c *conversation) record(v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = c.Write(append(raw, '\n'))
	return err
}
func (c *conversation) accept(body string) error {
	c.turn++
	timestamp := func() string { return time.Now().UTC().Format(time.RFC3339Nano) }
	reply := "Fixture completed the submitted input."
	if c.agent == "claude" {
		for _, m := range []struct{ role, text string }{{"user", body}, {"assistant", reply}} {
			if err := c.record(map[string]any{"type": m.role, "timestamp": timestamp(), "sessionId": c.id, "isSidechain": false, "entrypoint": "cli", "cwd": c.cwd, "message": map[string]any{"role": m.role, "content": []map[string]string{{"type": "text", "text": m.text}}}}); err != nil {
				return err
			}
		}
		return nil
	}
	turnID := fmt.Sprintf("fixture-%d-%d", os.Getpid(), c.turn)
	if err := c.record(map[string]any{"timestamp": timestamp(), "type": "event_msg", "payload": map[string]string{"type": "task_started", "turn_id": turnID}}); err != nil {
		return err
	}
	for _, m := range []struct{ role, kind, text string }{{"user", "input_text", body}, {"assistant", "output_text", reply}} {
		if err := c.record(map[string]any{"timestamp": timestamp(), "type": "response_item", "payload": map[string]any{"type": "message", "role": m.role, "content": []map[string]string{{"type": m.kind, "text": m.text}}}}); err != nil {
			return err
		}
	}
	return c.record(map[string]any{"timestamp": timestamp(), "type": "event_msg", "payload": map[string]string{"type": "task_complete", "turn_id": turnID}})
}
