package dendrogram

import (
	"math"
	"slices"
)

// Linkage is how far apart two clusters are, from the distances between
// their items.
type Linkage int

const (
	// Single is the distance of their nearest items, which chains.
	Single Linkage = iota
	// Complete is the distance of their farthest items.
	Complete
	// Average is the mean distance of their items, UPGMA.
	Average
	// Ward is how much merging them grows the variance within clusters,
	// meant for Euclidean distances.
	Ward
)

func (l Linkage) String() string {
	switch l {
	case Single:
		return "single"
	case Complete:
		return "complete"
	case Average:
		return "average"
	case Ward:
		return "Ward"
	}
	return "unknown"
}

// Metric is the distance between two items' vectors.
type Metric int

const (
	// Euclidean is the straight-line distance.
	Euclidean Metric = iota
	// Cosine is 1 less the cosine of their angle, from 0 for vectors that
	// point the same way to 2 for opposite ones, whatever their lengths.
	Cosine
)

// Tree is the outcome of clustering items: the merges that join them, two
// clusters at a time, the nearest first, up to one cluster of all.
type Tree struct {
	link   Linkage
	n      int
	merges []merge
	// order is the items as the leaves line up, each cluster's together,
	// so that no links cross.
	order []int
}

// merge joins clusters a and b at height: an item below n, or the cluster
// of merge k as n+k.
type merge struct {
	a, b   int
	height float64
	size   int
}

// Cluster clusters vectors, one for each item, by the distances metric
// finds between them, joined by link. It keeps an n by n matrix, and takes
// n³ steps: a few hundred items take milliseconds. Cluster once, as the
// data or the linkage change, not in each frame.
func Cluster(vectors [][]float64, metric Metric, link Linkage) *Tree {
	n := len(vectors)
	d := make([][]float64, n)
	for i := range d {
		d[i] = make([]float64, n)
		for j := range i {
			d[i][j] = distance(vectors[i], vectors[j], metric)
			d[j][i] = d[i][j]
		}
	}
	return cluster(d, link)
}

// ClusterDistances clusters items by d, n by n, the distance from each
// item to each other, joined by link, as Cluster does.
func ClusterDistances(d [][]float64, link Linkage) *Tree {
	c := make([][]float64, len(d))
	for i, row := range d {
		c[i] = slices.Clone(row)
	}
	return cluster(c, link)
}

func distance(a, b []float64, metric Metric) float64 {
	n := min(len(a), len(b))
	if metric == Cosine {
		var dot, na, nb float64
		for k := range n {
			dot += a[k] * b[k]
			na += a[k] * a[k]
			nb += b[k] * b[k]
		}
		if na == 0 || nb == 0 {
			return 1
		}
		return 1 - dot/math.Sqrt(na*nb)
	}
	var sum float64
	for k := range n {
		sum += (a[k] - b[k]) * (a[k] - b[k])
	}
	return math.Sqrt(sum)
}

// cluster joins the nearest two clusters until one is left, updating the
// distances to the cluster they make by Lance and Williams' formula. It
// takes d for its own.
func cluster(d [][]float64, link Linkage) *Tree {
	n := len(d)
	t := &Tree{link: link, n: n}
	if link == Ward {
		// Ward's update holds for squared distances, and its heights are
		// their roots, as SciPy's are.
		for _, row := range d {
			for j := range row {
				row[j] *= row[j]
			}
		}
	}
	id := make([]int, n)
	size := make([]int, n)
	alive := make([]int, n)
	for i := range n {
		id[i], size[i], alive[i] = i, 1, i
	}
	heights := make([]float64, 2*n)
	for range max(0, n-1) {
		bi, bj, best := 0, 0, math.Inf(1)
		for x, i := range alive {
			for _, j := range alive[x+1:] {
				if d[i][j] < best {
					bi, bj, best = i, j, d[i][j]
				}
			}
		}
		// Distances of NaN or infinity leave no pair nearest.
		if math.IsInf(best, 1) {
			bi, bj, best = alive[0], alive[1], 0
		}
		h := best
		if link == Ward {
			h = math.Sqrt(max(0, h))
		}
		// Rounding can put a merge a hair below one it contains, which
		// would draw a link below its own children.
		h = max(h, heights[id[bi]], heights[id[bj]])
		ni, nj := float64(size[bi]), float64(size[bj])
		for _, k := range alive {
			if k == bi || k == bj {
				continue
			}
			dik, djk, nk := d[bi][k], d[bj][k], float64(size[k])
			var v float64
			switch link {
			case Single:
				v = min(dik, djk)
			case Complete:
				v = max(dik, djk)
			case Average:
				v = (ni*dik + nj*djk) / (ni + nj)
			case Ward:
				v = ((ni+nk)*dik + (nj+nk)*djk - nk*best) / (ni + nj + nk)
			}
			d[bi][k], d[k][bi] = v, v
		}
		heights[n+len(t.merges)] = h
		t.merges = append(t.merges, merge{a: id[bi], b: id[bj], height: h, size: size[bi] + size[bj]})
		id[bi], size[bi] = n+len(t.merges)-1, size[bi]+size[bj]
		alive = slices.DeleteFunc(alive, func(k int) bool { return k == bj })
	}
	t.order = make([]int, 0, n)
	var walk func(node int)
	walk = func(node int) {
		if node < n {
			t.order = append(t.order, node)
			return
		}
		m := t.merges[node-n]
		walk(m.a)
		walk(m.b)
	}
	if n > 0 {
		walk(t.root())
	}
	return t
}

// root returns the node of all the items: the last merge, or the only
// item.
func (t *Tree) root() int {
	if len(t.merges) == 0 {
		return 0
	}
	return t.n + len(t.merges) - 1
}

// height returns the height of node, 0 for an item.
func (t *Tree) height(node int) float64 {
	if node < t.n {
		return 0
	}
	return t.merges[node-t.n].height
}

// Cut returns the height that cuts the tree into k clusters: midway
// between the merges that make k and k-1 of them. Merges at the same
// height can make fewer.
func (t *Tree) Cut(k int) float64 {
	m := len(t.merges)
	if m == 0 {
		return 0
	}
	k = max(1, min(t.n, k))
	if k == 1 {
		return t.merges[m-1].height * 1.05
	}
	hi := t.merges[t.n-k].height
	if k == t.n {
		return hi / 2
	}
	return (t.merges[t.n-k-1].height + hi) / 2
}

// Clusters returns the cluster of each item, cutting the tree at height
// cut, and how many clusters there are. Clusters are numbered from 0 in
// the order the leaves show them.
func (t *Tree) Clusters(cut float64) (of []int, n int) {
	of = make([]int, t.n)
	if t.n == 0 {
		return of, 0
	}
	var mark func(node, k int)
	mark = func(node, k int) {
		if node < t.n {
			of[node] = k
			return
		}
		m := t.merges[node-t.n]
		mark(m.a, k)
		mark(m.b, k)
	}
	var split func(node int)
	split = func(node int) {
		if node >= t.n && t.height(node) > cut {
			m := t.merges[node-t.n]
			split(m.a)
			split(m.b)
			return
		}
		mark(node, n)
		n++
	}
	split(t.root())
	return of, n
}
