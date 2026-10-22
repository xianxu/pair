package couchcmd

import "fmt"

const editorEndpointUsage = "couch --editor-endpoint register|unregister --json (JSON on stdin); lookup --namespace PATH --document-id ID --json"

func parseEditorEndpointCLI(args []string) (cliInvocation, error) {
	invalid := func() (cliInvocation, error) { return cliInvocation{}, fmt.Errorf("usage: %s", editorEndpointUsage) }
	if len(args) < 3 {
		return invalid()
	}
	verb := args[1]
	if verb == "register" || verb == "unregister" {
		if len(args) != 3 || args[2] != "--json" {
			return invalid()
		}
		return cliInvocation{kind: cliEditorEndpoint, args: []string{verb}}, nil
	}
	if verb != "lookup" {
		return invalid()
	}
	values := map[string]string{}
	for i := 2; i < len(args); i++ {
		key := args[i]
		if _, ok := values[key]; ok {
			return invalid()
		}
		switch key {
		case "--json":
			values[key] = "yes"
		case "--namespace", "--document-id":
			if i+1 >= len(args) {
				return invalid()
			}
			i++
			values[key] = args[i]
		default:
			return invalid()
		}
	}
	if values["--json"] == "" || values["--namespace"] == "" || values["--document-id"] == "" {
		return invalid()
	}
	return cliInvocation{kind: cliEditorEndpoint, args: []string{verb, values["--namespace"], values["--document-id"]}}, nil
}
