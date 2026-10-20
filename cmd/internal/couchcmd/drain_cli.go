package couchcmd

import (
	"fmt"
	"github.com/xianxu/pair/cmd/internal/couchcore"
	"github.com/xianxu/pair/cmd/internal/couchmessage"
	"time"
)

func parseDrainCLI(args []string) (cliInvocation, error) {
	d := &couchmessage.DrainRequest{Action: "start", TimeoutMillis: 600000}
	inv := cliInvocation{kind: cliMessage, messageOp: "drain", drain: d}
	seen := map[string]bool{}
	mode := ""
	for i := 1; i < len(args); i++ {
		arg := args[i]
		if seen[arg] {
			return cliInvocation{}, fmt.Errorf("duplicate drain option %s", arg)
		}
		seen[arg] = true
		value := func() (string, error) {
			if i+1 >= len(args) {
				return "", fmt.Errorf("%s requires a value", arg)
			}
			i++
			return args[i], nil
		}
		switch arg {
		case "--json":
			inv.jsonOutput = true
		case "--fast":
			d.Fast = true
		case "--reason":
			v, e := value()
			if e != nil {
				return cliInvocation{}, e
			}
			d.Reason = v
		case "--timeout":
			v, e := value()
			if e != nil {
				return cliInvocation{}, e
			}
			dur, e := time.ParseDuration(v)
			if e != nil || dur < time.Second || dur > 24*time.Hour || dur%time.Millisecond != 0 {
				return cliInvocation{}, fmt.Errorf("drain timeout must be 1s through 24h")
			}
			d.TimeoutMillis = dur.Milliseconds()
		case "--status", "--cancel", "--ack":
			if mode != "" {
				return cliInvocation{}, fmt.Errorf("drain takes one action")
			}
			mode = arg
			d.Action = arg[2:]
			if arg == "--ack" {
				v, e := value()
				if e != nil {
					return cliInvocation{}, e
				}
				if _, e = couchcore.DrainNotice(v, "", false); e != nil {
					return cliInvocation{}, e
				}
				d.ID = v
			}
		default:
			return cliInvocation{}, fmt.Errorf("unknown drain option %q", arg)
		}
	}
	if mode != "" {
		if seen["--reason"] || seen["--fast"] || seen["--timeout"] {
			return cliInvocation{}, fmt.Errorf("drain %s cannot carry start options", mode)
		}
		d.TimeoutMillis = 0
	} else if _, e := couchcore.DrainNotice("00000000-0000-4000-8000-000000000001", d.Reason, d.Fast); e != nil {
		return cliInvocation{}, e
	}
	return inv, nil
}
