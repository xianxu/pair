package couchcmd

import (
	"maps"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
)

// The attention list's sources (pair#673) report facts to the Console, which
// owns the list: the hub's notifications (handleNotify), the prompt set
// (publishBlocked) and the question set (the wait watcher).

// SetAttention installs the Console's attention observer. It must not block.
func (s *messageService) SetAttention(observe func(couchcore.AttentionFact)) {
	s.attention.Store(&observe)
}

func (s *messageService) observeAttention(f couchcore.AttentionFact) {
	if observe := s.attention.Load(); observe != nil && *observe != nil {
		(*observe)(f)
	}
}

// bindingThread is the thread a binding's wrapper serves.
func bindingThread(b couchmessage.Binding) couchcore.ThreadAddress {
	return couchcore.ThreadAddress{RepoScope: b.Scope, Tag: couchcore.ThreadTag(b.Tag)}
}

// attentionPromptTitle is what a prompt item says: the wrapper reports that a
// picker is up, not its text.
const attentionPromptTitle = "waiting on a prompt"

// questionEpisode reports the question an episode judged, if it judged one.
func questionEpisode(ep waitEpisode) (string, bool) {
	if !ep.notified {
		return "", false
	}
	reason, summary, _ := strings.Cut(ep.judged, "\x00")
	return summary, reason == string(waitQuestion)
}

// publishQuestions hands the Console the slots in a current question episode
// when that set changed (or on the first tick, so a restart reconciles).
func (w *waitWatcher) publishQuestions(live map[couchcore.ThreadAddress]couchcore.AttentionCondition) {
	if w.questions != nil && maps.Equal(live, w.questions) {
		return
	}
	w.questions = live
	w.svc.observeAttention(couchcore.AttentionFact{Kind: couchcore.AttentionQuestion, Live: live})
}
