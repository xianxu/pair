package webcast

func (c *Cast) SubmissionCandidateUpdates() (<-chan struct{}, func()) {
	if c.board == nil {
		ch := make(chan struct{})
		close(ch)
		return ch, func() {}
	}
	return c.board.SubscribeSubmissionCandidates()
}
