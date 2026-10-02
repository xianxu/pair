package couchcmd

import (
	"encoding/json"
	"fmt"
	"io"
)

func runAdoptionCLI(inv cliInvocation, rt Runtime, stdout, stderr io.Writer) int {
	production, ok := rt.(OSRuntime)
	if !ok {
		fmt.Fprintln(stderr, "couch: adoption requires an OS runtime")
		return 1
	}
	manager, request, _, err := production.singletonManager()
	if err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	a := inv.adoption
	for _, field := range []struct {
		value  string
		target *string
	}{{a.StoreDir, &request.Roots.StoreDir}, {a.PairDataDir, &request.Roots.PairDataDir}, {a.IdentityDir, &request.Roots.IdentityDir}} {
		if field.value != "" {
			*field.target, err = canonicalFuture(field.value)
			if err != nil {
				fmt.Fprintln(stderr, "couch:", err)
				return 1
			}
		}
	}
	for _, group := range []struct {
		values []string
		target *[]string
	}{{a.Stores, &request.Stores}, {a.Exclude, &request.Exclude}} {
		for _, value := range group.values {
			path, e := canonicalFuture(value)
			if e != nil {
				fmt.Fprintln(stderr, "couch:", e)
				return 1
			}
			*group.target = append(*group.target, path)
		}
	}
	if a.Expect != "" {
		selection, err := manager.Adopt(request, a.Expect)
		if err != nil {
			fmt.Fprintln(stderr, "couch:", err)
			return 1
		}
		if err = json.NewEncoder(stdout).Encode(struct {
			Status    string `json:"status"`
			Selection any    `json:"selection"`
		}{"SELECTED", selection}); err != nil {
			fmt.Fprintln(stderr, "couch:", err)
			return 1
		}
		return 0
	}
	report, err := manager.Preview(request)
	if err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	if err = json.NewEncoder(stdout).Encode(report); err != nil {
		fmt.Fprintln(stderr, "couch:", err)
		return 1
	}
	if len(report.Blockers) > 0 {
		fmt.Fprintln(stderr, "couch: UNMIGRATED; preserve the reported stores and resolve the blockers before applying adoption")
		return 1
	}
	return 0
}
