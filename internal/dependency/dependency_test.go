package dependency

import (
	"os"
	"testing"
)

const sampleCompose = `
services:
  web:
    depends_on:
      - api
      - db
  api:
    depends_on:
      - db
  db:
    image: postgres:latest
  cache:
    depends_on:
      - db
`

const cycleCompose = `
services:
  a:
    depends_on:
      - b
  b:
    depends_on:
      - c
  c:
    depends_on:
      - a
`

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp("", "compose-*.yml")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		t.Fatalf("failed to write temp file: %v", err)
	}
	f.Close()
	return f.Name()
}

func TestBuildDependencyGraph(t *testing.T) {
	file := writeTempFile(t, sampleCompose)
	defer os.Remove(file)
	mgr, err := NewDependencyManager(file)
	if err != nil {
		t.Fatalf("failed to create manager: %v", err)
	}
	graph, err := mgr.BuildDependencyGraph(file)
	if err != nil {
		t.Fatalf("failed to build graph: %v", err)
	}
	if len(graph.Services) != 4 {
		t.Errorf("expected 4 services, got %d", len(graph.Services))
	}
	if len(graph.Services["web"]) != 2 || graph.Services["web"][0] != "api" || graph.Services["web"][1] != "db" {
		t.Errorf("web dependencies incorrect: %+v", graph.Services["web"])
	}
	if len(graph.Services["api"]) != 1 || graph.Services["api"][0] != "db" {
		t.Errorf("api dependencies incorrect: %+v", graph.Services["api"])
	}
	if len(graph.Services["db"]) != 0 {
		t.Errorf("db should have no dependencies, got: %+v", graph.Services["db"])
	}
}

func TestGetUpdateOrder(t *testing.T) {
	file := writeTempFile(t, sampleCompose)
	defer os.Remove(file)
	mgr, _ := NewDependencyManager(file)
	order, err := mgr.GetUpdateOrder([]string{"web", "api", "db", "cache"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// db must come before api, web, and cache; api before web
	idx := func(s string) int {
		for i, v := range order {
			if v == s {
				return i
			}
		}
		return -1
	}
	if idx("db") > idx("api") || idx("db") > idx("web") || idx("db") > idx("cache") {
		t.Errorf("db should come before api, web, cache: %v", order)
	}
	if idx("api") > idx("web") {
		t.Errorf("api should come before web: %v", order)
	}
}

func TestGetUpdateOrder_Cycle(t *testing.T) {
	file := writeTempFile(t, cycleCompose)
	defer os.Remove(file)
	mgr, _ := NewDependencyManager(file)
	_, err := mgr.GetUpdateOrder([]string{"a", "b", "c"})
	if err == nil || err.Error() == "" {
		t.Error("expected error for circular dependency, got nil")
	}
}

func TestGetDependents(t *testing.T) {
	file := writeTempFile(t, sampleCompose)
	defer os.Remove(file)
	mgr, _ := NewDependencyManager(file)
	deps, err := mgr.GetDependents("db")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// web, api, and cache all depend (directly or indirectly) on db
	found := map[string]bool{}
	for _, d := range deps {
		found[d] = true
	}
	for _, want := range []string{"web", "api", "cache"} {
		if !found[want] {
			t.Errorf("expected %s to be a dependent of db", want)
		}
	}
}

func TestShouldUpdateDependents(t *testing.T) {
	file := writeTempFile(t, sampleCompose)
	defer os.Remove(file)
	mgr, _ := NewDependencyManager(file)
	if !mgr.ShouldUpdateDependents("db", "update") {
		t.Error("expected db to have dependents needing update")
	}
	if mgr.ShouldUpdateDependents("web", "update") {
		t.Error("expected web to have no dependents needing update")
	}
}
