package keywords

import (
	"math"
	"slices"
)

func scoreTextRank(stems []string) []Keyword {
	nodes, _, edges := buildGraph(stems)
	scores := pagerank(nodes, edges)

	freq := make(map[string]int, len(nodes))
	for _, s := range stems {
		freq[s]++
	}

	result := make([]Keyword, len(nodes))
	for i, node := range nodes {
		result[i] = Keyword{Stem: node, Score: scores[i], Count: freq[node]}
	}
	return result
}

// edge is a neighbor index + weight pair used for deterministic iteration.
type edge struct {
	to     int
	weight float64
}

func buildGraph(stems []string) (nodes []string, index map[string]int, edges [][]edge) {
	index = make(map[string]int)
	for _, s := range stems {
		if _, ok := index[s]; !ok {
			index[s] = len(nodes)
			nodes = append(nodes, s)
		}
	}

	// Accumulate in maps, then convert to sorted slices.
	edgeMaps := make([]map[int]float64, len(nodes))
	for i := range edgeMaps {
		edgeMaps[i] = make(map[int]float64)
	}

	for i, s := range stems {
		si := index[s]
		end := min(i+textrankWindowSize, len(stems))
		for j := i + 1; j < end; j++ {
			sj := index[stems[j]]
			if si != sj {
				edgeMaps[si][sj]++
				edgeMaps[sj][si]++
			}
		}
	}

	edges = make([][]edge, len(nodes))
	for i, m := range edgeMaps {
		edges[i] = make([]edge, 0, len(m))
		for to, w := range m {
			edges[i] = append(edges[i], edge{to: to, weight: w})
		}
		slices.SortFunc(edges[i], func(a, b edge) int {
			return a.to - b.to
		})
	}

	return nodes, index, edges
}

func pagerank(nodes []string, edges [][]edge) []float64 {
	n := len(nodes)
	if n == 0 {
		return nil
	}

	scores := make([]float64, n)
	for i := range scores {
		scores[i] = 1.0 / float64(n)
	}

	outWeight := make([]float64, n)
	for i, neighbors := range edges {
		for _, e := range neighbors {
			outWeight[i] += e.weight
		}
	}

	nf := float64(n)
	for range textrankMaxIter {
		newScores := make([]float64, n)
		maxDelta := 0.0

		for i := range n {
			sum := 0.0
			for _, e := range edges[i] {
				if outWeight[e.to] > 0 {
					sum += (e.weight / outWeight[e.to]) * scores[e.to]
				}
			}
			newScores[i] = (1-textrankDamping)/nf + textrankDamping*sum
			delta := math.Abs(newScores[i] - scores[i])
			if delta > maxDelta {
				maxDelta = delta
			}
		}

		scores = newScores
		if maxDelta < textrankEpsilon {
			break
		}
	}

	return scores
}
