package couchcmd

import _ "embed"

// couchSkill is the single shipped skill source, available without a running
// supervisor so an operator can load or install it before opening a session.
//
//go:embed skills/couch/SKILL.md
var couchSkill string
