package pr

import (
	"sort"
	"strconv"
)

// Stacks groups pull requests into graphite stacks. A stack is the connected
// component of the base -> head graph: anything based on a branch with no pull
// request of its own (the default branch, or an unpushed base) ends the chain.
//
// Nothing here is stored. Restacking rewrites every link, so a stack is
// recomputed from the base and head of each pull request whenever it is asked
// for.
type Stacks struct {
	prs    []PR
	byHead map[string]int // repo#head -> index into prs
	index  map[string]int // repo#number -> index into prs
	parent []int
}

func headKey(repo, head string) string { return repo + "#" + head }

func (p PR) numberKey() string { return p.Repo + "##" + strconv.Itoa(p.Number) }

func BuildStacks(prs []PR) *Stacks {
	s := &Stacks{
		prs:    prs,
		byHead: map[string]int{},
		index:  map[string]int{},
		parent: make([]int, len(prs)),
	}
	for i, p := range prs {
		s.parent[i] = i
		s.index[p.numberKey()] = i
	}

	// A branch can carry an old closed pull request as well as a live one, so
	// an open pull request always wins the branch, then the newest number.
	ranked := make([]int, len(prs))
	for i := range ranked {
		ranked[i] = i
	}
	sort.SliceStable(ranked, func(a, b int) bool {
		pa, pb := prs[ranked[a]], prs[ranked[b]]
		if (pa.State == "OPEN") != (pb.State == "OPEN") {
			return pa.State == "OPEN"
		}
		return pa.Number > pb.Number
	})
	for _, i := range ranked {
		key := headKey(prs[i].Repo, prs[i].Head)
		if _, taken := s.byHead[key]; !taken {
			s.byHead[key] = i
		}
	}

	for i, p := range prs {
		if below, ok := s.byHead[headKey(p.Repo, p.Base)]; ok {
			s.parent[s.find(i)] = s.find(below)
		}
	}
	return s
}

func (s *Stacks) find(i int) int {
	for s.parent[i] != i {
		s.parent[i] = s.parent[s.parent[i]]
		i = s.parent[i]
	}
	return i
}

// ByHead is the pull request a branch carries.
func (s *Stacks) ByHead(repo, head string) (PR, bool) {
	i, ok := s.byHead[headKey(repo, head)]
	if !ok {
		return PR{}, false
	}
	return s.prs[i], true
}

// BottomOf is the pull request at the foot of p's stack. Nothing above the
// bottom can merge until the bottom does, so it is the one worth reporting.
// Found by following base links rather than taking the lowest number, because
// restacking reorders a stack.
func (s *Stacks) BottomOf(p PR) PR {
	seen := map[int]bool{}
	if i, ok := s.index[p.numberKey()]; ok {
		seen[i] = true
	}
	for {
		below, ok := s.byHead[headKey(p.Repo, p.Base)]
		if !ok || seen[below] {
			return p
		}
		seen[below] = true
		p = s.prs[below]
	}
}

// StackOf is the numbers of every pull request in p's stack, ascending.
func (s *Stacks) StackOf(p PR) []int {
	i, ok := s.index[p.numberKey()]
	if !ok {
		return []int{p.Number}
	}
	root := s.find(i)
	var numbers []int
	for j := range s.prs {
		if s.find(j) == root {
			numbers = append(numbers, s.prs[j].Number)
		}
	}
	sort.Ints(numbers)
	return numbers
}
