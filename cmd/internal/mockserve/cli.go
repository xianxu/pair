package mockserve

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Command is the closed public mock command, independent of Couch runtime.
type Command struct {
	Mode, Dir, Version string
	Port               int
}

func ParseCommand(args []string) (Command, error) {
	usage := func() (Command, error) {
		return Command{}, fmt.Errorf("use couch mock serve DIR [--port N] or couch mock use DIR mock-N")
	}
	if len(args) == 0 || len(args) == 1 && (args[0] == "--help" || args[0] == "-h") {
		return Command{Mode: "help"}, nil
	}
	if len(args) < 2 || strings.TrimSpace(args[1]) == "" {
		return usage()
	}
	c := Command{Mode: args[0], Dir: args[1]}
	switch c.Mode {
	case "serve":
		if len(args) == 2 {
			return c, nil
		}
		if len(args) != 4 || args[2] != "--port" {
			return usage()
		}
		n, err := strconv.Atoi(args[3])
		if err != nil || n < 0 || n > 65535 {
			return Command{}, fmt.Errorf("port must be an integer from 0 to 65535")
		}
		c.Port = n
	case "use":
		if len(args) != 3 {
			return usage()
		}
		if _, err := ParseVersion(args[2]); err != nil {
			return Command{}, err
		}
		c.Version = args[2]
	default:
		return usage()
	}
	return c, nil
}
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	c, err := ParseCommand(args)
	if err != nil {
		fmt.Fprintln(stderr, "couch mock:", err)
		return 2
	}
	if c.Mode == "help" {
		fmt.Fprintln(stdout, "couch mock serve DIR [--port N]\ncouch mock use DIR mock-N\n\nServe version folders on 127.0.0.1, with shared fixtures.json and live reload.\nDefault: highest numbered version. use persists an explicit selection in .mock-current.\nCtrl-C stops serving; no browser is opened.")
		return 0
	}
	if c.Mode == "serve" {
		err = Serve(ctx, c.Dir, c.Port, stdout)
	} else {
		var p *Project
		p, err = OpenProject(c.Dir)
		if err == nil {
			err = p.Use(c.Version)
			closeErr := p.Close()
			if err == nil {
				err = closeErr
			}
		}
		if err == nil {
			_, err = fmt.Fprintln(stdout, "Selected", c.Version)
		}
	}
	if err != nil {
		fmt.Fprintln(stderr, "couch mock:", err)
		return 1
	}
	return 0
}
