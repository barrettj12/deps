package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Usage:
//   deps tree <root>
//   deps path <root> <target>
//   deps paths <root> <target>
func main() {
	if len(os.Args) <= 1 {
		fmt.Println(`
no action specified
valid actions are: tree, path, paths`[1:])
		os.Exit(1)
	}

	action := os.Args[1]
	root := os.Args[2]
	switch action {
	case "tree":
		fmt.Printf("Building dependency tree for %q...\n", root)
		t, err := buildTree(root)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		t.print("")

	case "path":
		target := os.Args[3]
		fmt.Printf("Finding path from %q to %q...\n", root, target)
		p, err := findPath(root, target)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		for _, pkg := range p {
			fmt.Println(pkg)
		}

	case "paths":
		target := os.Args[3]
		fmt.Printf("Finding all paths from %q to %q...\n", root, target)
		paths, err := findAllPaths(root, target)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		for _, p := range paths {
			fmt.Println()
			for _, pkg := range p {
				fmt.Println(pkg)
			}
		}
		//case "help", "-help", "--help", "-h":
		//default:
	}
}

type path []string

func (p path) last() string {
	return p[len(p)-1]
}

func (p path) append(s string) path {
	return append(p, s)
}

// findPath import path from root to target
func findPath(root, target string) ([]string, error) {
	visited := set[string]{}
	q := queue[path]{}
	q.add(path{root})

	for !q.empty() {
		p := q.next()
		pkg := p.last()
		if visited.contains(pkg) {
			continue
		}
		visited.add(pkg)

		cmd := exec.Command("go", "list", "-f", "'{{.Imports}}'", pkg)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, err
		}

		deps := parseDeps(string(output))
		fdeps := filterDeps(deps)

		for _, dep := range fdeps {
			newPath := p.append(dep)
			if dep == target {
				return newPath, nil
			}
			q.add(newPath)
		}
	}

	return nil, fmt.Errorf("%q does not depend on %q", root, target)
}

// findAllPaths import path from root to target
func findAllPaths(root, target string) ([]path, error) {
	paths := []path{}
	visited := set[string]{}
	q := queue[path]{}
	q.add(path{root})

	for !q.empty() {
		p := q.next()
		pkg := p.last()
		fmt.Printf("%d    %s", q.size(), pkg)
		//if visited.contains(pkg) {
		//	continue
		//}
		//visited.add(pkg)

		cmd := exec.Command("go", "list", "-f", "'{{.Imports}}'", pkg)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, err
		}

		deps := parseDeps(string(output))
		fdeps := filterDeps(deps)

		for _, dep := range fdeps {
			// Remove duplicates
			newPath := p.append(dep)
			if dep == target {
				paths = append(paths, newPath)
			} else if !visited.contains(dep) {
				visited.add(dep)
				q.add(newPath)
			}
		}

		fmt.Print("\r%s", strings.Repeat(" ", len(pkg)+10))
		fmt.Print("\r")
	}

	return paths, nil
}

func parseDeps(raw string) []string {
	peeled := strings.Trim(raw, "'[]\n")
	return strings.Split(peeled, " ")
}

func filterDeps(deps []string) []string {
	fdeps := []string{}
	for _, dep := range deps {
		if strings.HasPrefix(dep, "github.com/juju/juju") {
			fdeps = append(fdeps, dep)
		}
	}
	return fdeps
}

type tree struct {
	val      string
	children []*tree
}

func (t *tree) print(prefix string) {
	fmt.Printf("%s%s\n", prefix, t.val)
	childPrefix := fmt.Sprintf("%s  ", prefix)
	for _, c := range t.children {
		c.print(childPrefix)
	}
}

// buildTree builds the dependency tree under root.
func buildTree(root string) (*tree, error) {
	base := &tree{root, nil}
	visited := set[string]{}
	q := queue[*tree]{}
	q.add(base)

	for !q.empty() {
		t := q.next()
		pkg := t.val
		fmt.Printf("%d    %s", q.size(), pkg)
		if visited.contains(pkg) {
			continue
		}
		visited.add(pkg)

		cmd := exec.Command("go", "list", "-f", "'{{.Imports}}'", pkg)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return nil, err
		}

		deps := parseDeps(string(output))
		fdeps := filterDeps(deps)

		for _, dep := range fdeps {
			depT := &tree{dep, nil}
			t.children = append(t.children, depT)
			q.add(depT)
		}
		fmt.Print("\r%s", strings.Repeat(" ", len(pkg)+10))
		fmt.Print("\r")
	}

	return base, nil
}

//// buildTree builds the dependency tree under root.
//func buildTree(root string) (*tree, error) {
//	base := &tree{root, nil}
//	visited := set[string]{}
//	q := queue[*tree]{}
//	q.add(base)
//
//	for !q.empty() {
//		t := q.next()
//		pkg := t.val
//		fmt.Print(pkg)
//		//if visited.contains(pkg) {
//		//	continue
//		//}
//		//visited.add(pkg)
//
//		cmd := exec.Command("go", "list", "-f", "'{{.Imports}}'", pkg)
//		output, err := cmd.CombinedOutput()
//		if err != nil {
//			return nil, err
//		}
//
//		deps := parseDeps(string(output))
//		fdeps := filterDeps(deps)
//
//		for _, dep := range fdeps {
//			// Remove duplicates
//			if !visited.contains(dep) {
//				visited.add(dep)
//				depT := &tree{dep, nil}
//				t.children = append(t.children, depT)
//				q.add(depT)
//			}
//		}
//		fmt.Print("\r%s", strings.Repeat(" ", len(pkg)))
//		fmt.Print("\r")
//	}
//
//	return base, nil
//}

// Queue implementation
type queue[T any] struct {
	vals []T
}

func (q *queue[T]) empty() bool {
	return len(q.vals) == 0
}

func (q *queue[T]) add(t T) {
	q.vals = append(q.vals, t)
}

func (q *queue[T]) next() T {
	next := q.vals[0]
	q.vals = q.vals[1:]
	return next
}

func (q *queue[T]) size() int {
	return len(q.vals)
}

// Set implementation
type set[T comparable] map[T]struct{}

func (s *set[T]) add(t T) {
	(*s)[t] = struct{}{}
}

func (s *set[T]) contains(t T) bool {
	_, ok := (*s)[t]
	return ok
}
