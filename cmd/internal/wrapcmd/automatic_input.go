package wrapcmd

// All fields except automaticRenderEpoch are owned by the single stdin writer.
// The admission mutex fences human input; this lease fences automatic writers
// across separate paste/render/submit callbacks.
type automaticInputPhase uint8

const (
	automaticInputIdle automaticInputPhase = iota
	automaticInputOrientation
	automaticInputPeer
	automaticInputAwaitingEmpty
)

type automaticInputTransaction struct {
	phase       automaticInputPhase
	renderFence uint64
}

func (p *proxy) automaticInputAvailable(owner automaticInputPhase) bool {
	tx := &p.automaticInput
	if tx.phase == automaticInputAwaitingEmpty {
		if p.automaticRenderEpoch.Load() <= tx.renderFence || p.terminal == nil ||
			peerComposerState(p.agentBasename, p.terminal.Snapshot()) != PeerComposerEmpty {
			return false
		}
		tx.phase = automaticInputIdle
	}
	return tx.phase == automaticInputIdle || tx.phase == owner
}

func (p *proxy) finishAutomaticInput(owner automaticInputPhase) {
	if p.automaticInput.phase == owner {
		p.automaticInput = automaticInputTransaction{
			phase:       automaticInputAwaitingEmpty,
			renderFence: p.automaticRenderEpoch.Load(),
		}
	}
}
