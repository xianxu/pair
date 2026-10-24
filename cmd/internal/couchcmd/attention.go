package couchcmd

import (
	"fmt"
	"maps"
	"strings"

	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"github.com/xianxu/pair/cmd/internal/couchtty"
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

// noticeCondition identifies an episode by its exact source, never its label.
func noticeCondition(class couchcore.NoticeClass, thread couchcore.ThreadAddress) string {
	return fmt.Sprintf("%s\x00%s\x00%s", class, thread.RepoScope, thread.Tag)
}

// publishQuestions reconciles human eligibility on every owner snapshot, even
// while the judged episode itself is unchanged. The wait watcher remains the
// sole owner of its canonical hub send.
func (w *waitWatcher) publishQuestions(live map[couchcore.ThreadAddress]couchcore.AttentionCondition) {
	eligible := map[couchcore.ThreadAddress]couchcore.AttentionCondition{}
	for thread, condition := range live {
		owner := w.svc.NoticeOwner(couchcore.NoticeDeliveryOccurrence{Thread: thread})
		if couchcore.RouteNotice(couchcore.NoticeRouteInput{Class: couchcore.NoticeJudgedQuestion, Owner: owner.State, HubHandled: true}).Bell {
			eligible[thread] = condition
		}
		if previous, existed := w.questionConditions[thread]; !existed || previous != condition {
			w.svc.publishSystemNotice(couchtty.Notice{Class: couchcore.NoticeJudgedQuestion, Kind: "status", Thread: thread, Label: condition.Label, Body: condition.Title, Condition: noticeCondition(couchcore.NoticeJudgedQuestion, thread), HubHandled: true})
		}
	}
	for thread := range w.questionConditions {
		if _, exists := live[thread]; !exists {
			w.svc.publishSystemNotice(couchtty.Notice{Class: couchcore.NoticeJudgedQuestion, Thread: thread, Condition: noticeCondition(couchcore.NoticeJudgedQuestion, thread), Cleared: true})
		}
	}
	w.questionConditions = live
	if w.questions != nil && maps.Equal(eligible, w.questions) {
		return
	}
	w.questions = eligible
	w.svc.observeAttention(couchcore.AttentionFact{Kind: couchcore.AttentionQuestion, Live: eligible})
}
