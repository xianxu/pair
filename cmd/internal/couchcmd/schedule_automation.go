package couchcmd

// scheduleAutomationOn is the service-owned admission seam. The concrete
// automation getter is wired by the second lander of #737 and #739.
func (s *messageService) scheduleAutomationOn(root string) bool {
	return s.scheduleAutomationEnabled == nil || s.scheduleAutomationEnabled(root)
}

func (o *scheduleOwner) automationOn(root string) bool {
	return o.automationEnabled == nil || o.automationEnabled(root)
}
