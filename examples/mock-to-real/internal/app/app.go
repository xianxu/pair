// Package app is the real application. It never imports development tooling.
package app

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"strings"
)

// Screen is the production projection: no mock paths or fixtures.
type Screen struct {
	ID    string `json:"id"`
	Route string `json:"route"`
	State string `json:"state"`
}
type Run struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

var runs = []Run{{"live-42", "Production health check", "Complete"}, {"live-43", "Nightly archive", "Running"}}

// RealHandlers is the sole registration map. Implement a page here before marking it real.
func RealHandlers() map[string]http.HandlerFunc {
	return map[string]http.HandlerFunc{
		"index": func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "<h1>Real overview</h1><p>Independent application data.</p>")
		},
		"runs": func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprint(w, "<h1>Real runs</h1>")
			for _, run := range runs {
				fmt.Fprintf(w, "<p>%s · %s</p>", html.EscapeString(run.Name), html.EscapeString(run.Status))
			}
		},
	}
}
func New(screens []Screen) (http.Handler, error) {
	handlers := RealHandlers()
	seenIDs := map[string]bool{}
	routes := make(map[string]Screen, len(screens))
	var navigation strings.Builder
	navigation.WriteString("<nav aria-label=\"Screens\">")
	for _, s := range screens {
		if s.ID == "" || !strings.HasPrefix(s.Route, "/") || strings.ContainsAny(s.Route, "?#") || seenIDs[s.ID] || routeExists(routes, s.Route) || s.Route == "/api/runs" || (s.State != "real" && s.State != "mock") {
			return nil, fmt.Errorf("configuration: invalid screen %q", s.ID)
		}
		seenIDs[s.ID] = true
		routes[s.Route] = s
		if s.State == "real" && handlers[s.ID] == nil {
			return nil, fmt.Errorf("configuration: screen %q marked real without handler", s.ID)
		}
		label := s.ID
		if s.State != "real" {
			label += " (Not implemented)"
		}
		fmt.Fprintf(&navigation, "<a href=\"%s\">%s</a> ", html.EscapeString(s.Route), html.EscapeString(label))
	}
	navigation.WriteString("</nav></body></html>")
	markup := navigation.String()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/runs" {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(runs)
			return
		}
		if s, ok := routes[r.URL.Path]; ok {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, "<!doctype html><html lang=\"en\"><meta charset=\"utf-8\"><title>Real application</title><body>")
			if s.State == "real" {
				handlers[s.ID](w, r)
			} else {
				fmt.Fprint(w, "<h1>Not implemented yet</h1><a href=\"/\">Back to overview</a>")
			}
			fmt.Fprint(w, markup)
			return
		}
		http.NotFound(w, r)
	}), nil
}

func routeExists(routes map[string]Screen, route string) bool { _, ok := routes[route]; return ok }
