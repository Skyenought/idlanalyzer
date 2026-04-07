package swagger2thrift

import (
	"sort"
	"strings"
)

// Graph represents a directed graph of dependencies (Grouped Namespaces).
type Graph struct {
	Nodes map[string][]string // Adjacency list: Node -> [Dependencies]
}

func newGraph() *Graph {
	return &Graph{
		Nodes: make(map[string][]string),
	}
}

func (g *Graph) AddEdge(from, to string) {
	if from == to {
		return
	}
	deps := g.Nodes[from]
	for _, d := range deps {
		if d == to {
			return // Already exists
		}
	}
	g.Nodes[from] = append(g.Nodes[from], to)
}

// analyzeDependency scans all schemas to build a file-level dependency graph
// and returns a mapping for merging files involved in circular dependencies.
func (c *Converter) analyzeDependency(schemas map[string]*Schema) map[string]string {
	graph := newGraph()
	// Map schema full name to its calculated file group (namespace)
	// Example: "a.b.c.Type" -> "a.b"
	schemaToGroup := make(map[string]string)

	// 1. Pre-calculate groups for all schemas to avoid repeated processing
	for name := range schemas {
		ns, _ := splitDefinitionName(name)
		if ns == "main" {
			schemaToGroup[name] = "main"
		} else {
			// Using depth 2 as defined in other parts of the project
			schemaToGroup[name] = getGroupedNamespace(ns, 2)
		}
	}

	// 2. Build dependency graph based on $ref
	// We are building a graph where nodes are "File Groups" (Namespaces)
	for name, schema := range schemas {
		currentGroup := schemaToGroup[name]
		refs := extractRefs(schema)

		for _, ref := range refs {
			defName := getDefinitionNameFromRef(ref)
			if targetGroup, ok := schemaToGroup[defName]; ok {
				// Add a directed edge: CurrentFile -> TargetFile
				graph.AddEdge(currentGroup, targetGroup)
			}
		}
	}

	// 3. Find Strongly Connected Components (SCCs)
	// Groups in the same SCC form a cycle and must be merged.
	sccs := tarjan(graph)

	// 4. Create mapping for merging
	// Map: original_group_name -> merged_group_name
	mapping := make(map[string]string)

	for _, scc := range sccs {
		if len(scc) > 1 {
			// Cycle detected! Pick a representative name.
			// Sort to ensure deterministic output.
			sort.Strings(scc)
			representative := scc[0]

			// If "main" is involved in the cycle, everything must merge into "main"
			for _, node := range scc {
				if node == "main" {
					representative = "main"
					break
				}
			}

			// In case the representative itself is very long (like your example),
			// you might want to consider hardcoding a shorter name or using the shortest one.
			// For now, we stick to the first alphabetically or "main".

			for _, node := range scc {
				mapping[node] = representative
			}
		}
	}

	return mapping
}

// extractRefs recursively finds all $ref strings in a schema
func extractRefs(s *Schema) []string {
	var refs []string
	if s == nil {
		return refs
	}
	if s.Ref != "" {
		refs = append(refs, s.Ref)
	}
	if s.Items != nil {
		refs = append(refs, extractRefs(s.Items)...)
	}
	for _, p := range s.Properties {
		refs = append(refs, extractRefs(p)...)
	}
	if s.AdditionalProperties != nil {
		if apSchema, ok := s.AdditionalProperties.(*Schema); ok {
			refs = append(refs, extractRefs(apSchema)...)
		}
		// Note: map[string]any case for AdditionalProperties is complex to parse refs from
		// without full recursion, skipping for simplicity as it's rare for refs.
	}
	for _, a := range s.AllOf {
		refs = append(refs, extractRefs(a)...)
	}
	return refs
}

func getDefinitionNameFromRef(ref string) string {
	if strings.HasPrefix(ref, "#/definitions/") {
		return strings.TrimPrefix(ref, "#/definitions/")
	}
	if strings.HasPrefix(ref, "#/components/schemas/") {
		return strings.TrimPrefix(ref, "#/components/schemas/")
	}
	return ref
}

// Tarjan's algorithm to find SCCs
func tarjan(g *Graph) [][]string {
	var index int
	var stack []string

	var nodes []string
	for n := range g.Nodes {
		nodes = append(nodes, n)
	}
	sort.Strings(nodes)

	indices := make(map[string]int)
	lowlink := make(map[string]int)
	onStack := make(map[string]bool)
	var sccs [][]string

	var strongconnect func(string)
	strongconnect = func(v string) {
		indices[v] = index
		lowlink[v] = index
		index++
		stack = append(stack, v)
		onStack[v] = true

		neighbors := g.Nodes[v]
		sort.Strings(neighbors) // Determinism

		for _, w := range neighbors {
			if _, visited := indices[w]; !visited {
				strongconnect(w)
				if lowlink[w] < lowlink[v] {
					lowlink[v] = lowlink[w]
				}
			} else if onStack[w] {
				if indices[w] < lowlink[v] {
					lowlink[v] = indices[w]
				}
			}
		}

		// If v is a root node, pop the stack and generate an SCC
		if lowlink[v] == indices[v] {
			var scc []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				scc = append(scc, w)
				if w == v {
					break
				}
			}
			sccs = append(sccs, scc)
		}
	}

	for _, node := range nodes {
		if _, visited := indices[node]; !visited {
			strongconnect(node)
		}
	}

	return sccs
}
