// Package sorting shows a sorting algorithm at work on a row of bars, one
// for each value, as it compares and moves them: the two bars it compares
// light up in the accent color and the ones it moves in orange, and once
// it is done the bars turn green from left to right. Above the bars, the
// algorithm's name and how many comparisons and moves it has made so far.
// Several views of different algorithms on the same values race them.
//
//	sorting.View(c, &app.quick, app.values, sorting.Quick, 120)
package sorting

import (
	"fmt"
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
)

// Algorithm is a sorting algorithm.
type Algorithm int

const (
	Bubble Algorithm = iota
	Insertion
	Selection
	Shell
	Merge
	Quick
	Heap
)

// Algorithms are all the algorithms, slowest first.
var Algorithms = []Algorithm{Bubble, Insertion, Selection, Shell, Merge, Quick, Heap}

func (a Algorithm) String() string {
	return [...]string{"Bubble sort", "Insertion sort", "Selection sort", "Shellsort", "Merge sort", "Quicksort", "Heapsort"}[a]
}

// OpKind is what a step of a sort does.
type OpKind int

const (
	// Compare compares the values at I and J.
	Compare OpKind = iota
	// Swap swaps the values at I and J.
	Swap
	// Write writes V at I.
	Write
)

// Op is a step of a sort.
type Op struct {
	Kind OpKind
	I, J int
	V    int
}

type recorder struct {
	a   []int
	ops []Op
}

func (r *recorder) less(i, j int) bool {
	r.ops = append(r.ops, Op{Kind: Compare, I: i, J: j})
	return r.a[i] < r.a[j]
}

func (r *recorder) swap(i, j int) {
	r.a[i], r.a[j] = r.a[j], r.a[i]
	r.ops = append(r.ops, Op{Kind: Swap, I: i, J: j})
}

func (r *recorder) write(i, v int) {
	r.a[i] = v
	r.ops = append(r.ops, Op{Kind: Write, I: i, V: v})
}

// Trace returns the steps algo takes to sort values, which it leaves as
// they are.
func Trace(algo Algorithm, values []int) []Op {
	r := &recorder{a: slices.Clone(values)}
	n := len(r.a)
	switch algo {
	case Bubble:
		for end := n - 1; end > 0; end-- {
			swapped := false
			for i := range end {
				if r.less(i+1, i) {
					r.swap(i, i+1)
					swapped = true
				}
			}
			if !swapped {
				break
			}
		}
	case Insertion:
		for i := 1; i < n; i++ {
			for j := i; j > 0 && r.less(j, j-1); j-- {
				r.swap(j, j-1)
			}
		}
	case Selection:
		for i := range n - 1 {
			m := i
			for j := i + 1; j < n; j++ {
				if r.less(j, m) {
					m = j
				}
			}
			if m != i {
				r.swap(i, m)
			}
		}
	case Shell:
		for _, gap := range []int{701, 301, 132, 57, 23, 10, 4, 1} {
			for i := gap; i < n; i++ {
				for j := i; j >= gap && r.less(j, j-gap); j -= gap {
					r.swap(j, j-gap)
				}
			}
		}
	case Merge:
		aux := make([]int, n)
		for width := 1; width < n; width *= 2 {
			for lo := 0; lo < n-width; lo += 2 * width {
				mid, hi := lo+width, min(lo+2*width, n)
				copy(aux[lo:hi], r.a[lo:hi])
				i, j := lo, mid
				for k := lo; k < hi; k++ {
					switch {
					case i >= mid:
						r.write(k, aux[j])
						j++
					case j >= hi:
						r.write(k, aux[i])
						i++
					default:
						r.ops = append(r.ops, Op{Kind: Compare, I: i, J: j})
						if aux[j] < aux[i] {
							r.write(k, aux[j])
							j++
						} else {
							r.write(k, aux[i])
							i++
						}
					}
				}
			}
		}
	case Quick:
		var sort func(lo, hi int)
		sort = func(lo, hi int) {
			if lo >= hi {
				return
			}
			r.swap((lo+hi)/2, hi)
			p := lo
			for i := lo; i < hi; i++ {
				if r.less(i, hi) {
					if i != p {
						r.swap(i, p)
					}
					p++
				}
			}
			r.swap(p, hi)
			sort(lo, p-1)
			sort(p+1, hi)
		}
		sort(0, n-1)
	case Heap:
		sift := func(root, end int) {
			for {
				child := 2*root + 1
				if child >= end {
					return
				}
				if child+1 < end && r.less(child, child+1) {
					child++
				}
				if !r.less(root, child) {
					return
				}
				r.swap(root, child)
				root = child
			}
		}
		for i := n/2 - 1; i >= 0; i-- {
			sift(i, n)
		}
		for end := n - 1; end > 0; end-- {
			r.swap(0, end)
			sift(0, end)
		}
	}
	return r.ops
}

// State is what a view keeps: the sort's steps and how far it has gone.
// Keep it in the app's state, one for each view.
type State struct {
	algo     Algorithm
	values   []int
	ops      []Op
	a        []int
	step     int
	compares int
	moves    int
	done     time.Time
	last     time.Time
}

// View creates the bars of values as algo sorts them, rate steps a
// second; a rate of 0 pauses it. New values or another algorithm start
// it over.
func View(c *ui.Context, s *State, values []int, algo Algorithm, rate float64) ui.Element {
	t := c.Theme()
	if s.ops == nil || s.algo != algo || !slices.Equal(s.values, values) {
		*s = State{algo: algo, values: slices.Clone(values), ops: Trace(algo, values), a: slices.Clone(values)}
	}
	now := c.Now()
	if rate <= 0 {
		s.last = time.Time{}
	} else if s.done.IsZero() {
		if s.last.IsZero() {
			s.last = now
		}
		n := int(now.Sub(s.last).Seconds() * rate)
		s.last = s.last.Add(time.Duration(float64(n) / rate * float64(time.Second)))
		for ; n > 0 && s.step < len(s.ops); n-- {
			s.apply(s.ops[s.step])
			s.step++
		}
		if s.step == len(s.ops) {
			s.done = now
		}
		c.After(time.Second / 60)
	}
	var cur Op
	active := s.done.IsZero() && s.step > 0
	if active {
		cur = s.ops[s.step-1]
	}
	// Once done, a sweep turns the bars green over 0.6 s.
	sweep := float32(0)
	if !s.done.IsZero() {
		sweep = min(1, float32(now.Sub(s.done).Seconds()/0.6))
		if sweep < 1 {
			c.After(time.Second / 60)
		}
	}
	top := 0
	for _, v := range s.a {
		top = max(top, v)
	}
	root := ui.Column(c).Gap(t.Space(2)).Role(ui.RoleGroup).Label(algo.String())
	root.Children(func() {
		ui.Row(c).Gap(t.Space(3)).AlignItems(ui.Baseline).Children(func() {
			ui.Text(c, algo.String()).FontWeight(600)
			ui.Text(c, fmt.Sprintf("%d comparisons · %d moves", s.compares, s.moves)).
				FontSize(t.FontSize - 1).TextColor(t.TextMuted).FontFeatures("tnum")
		})
		bars := ui.Box(c).Grow(1).Role(ui.RoleImage).Label(fmt.Sprintf("%d of %d steps", s.step, len(s.ops)))
		bars.Draw(func(p *ui.Painter, r ui.Rect) {
			n := len(s.a)
			if n == 0 || top <= 0 {
				return
			}
			w := r.W / float32(n)
			gap := float32(0)
			if w >= 4 {
				gap = max(1, w*0.15)
			}
			base := t.TextMuted.Alpha(0.45)
			green := ui.RGB(52, 178, 92)
			for i, v := range s.a {
				col := base
				switch {
				case float32(i) < sweep*float32(n):
					col = green
				case active && (i == cur.I || (cur.Kind != Write && i == cur.J)):
					col = t.Accent
					if cur.Kind != Compare {
						col = ui.RGB(240, 140, 40)
					}
				}
				h := r.H * float32(v) / float32(top)
				p.Fill(ui.Rect{X: r.X + float32(i)*w + gap/2, Y: r.Y + r.H - h, W: w - gap, H: h}, col, min(2, (w-gap)/2))
			}
		})
	})
	return root
}

func (s *State) apply(op Op) {
	switch op.Kind {
	case Compare:
		s.compares++
	case Swap:
		s.a[op.I], s.a[op.J] = s.a[op.J], s.a[op.I]
		s.moves++
	case Write:
		s.a[op.I] = op.V
		s.moves++
	}
}
