// Package thumbgrid shows a grid of thumbnails, as Photos and Finder's
// icon view do, made in the background as they scroll into view: a
// Loader makes each from the picture its Load returns, a few at once,
// those last shown first, and keeps the latest made.
//
//	app.thumbs = &thumbgrid.Loader[string]{Load: func(path string) (image.Image, error) {
//		f, err := os.Open(path)
//		...
//		img, _, err := image.Decode(f)
//		return img, err
//	}}
//	...
//	thumbgrid.View(c, &app.grid, app.thumbs, app.paths, 140, filepath.Base).Grow(1)
package thumbgrid

import (
	"container/list"
	"image"
	"runtime"
	"slices"
	"sync"

	"github.com/egoist/mygo/ui"
	"golang.org/x/image/draw"
)

// maxQueue is how many thumbnails wait to be made at most; asked for
// more, the loader forgets those asked for longest ago, scrolled past.
const maxQueue = 256

// Loader makes thumbnails of the pictures of keys in the background, and
// keeps the latest made. Its fields are read once, at the first Thumb;
// its methods are for the main thread.
type Loader[K comparable] struct {
	// Load returns the picture of key, called on Workers goroutines at
	// once.
	Load func(key K) (image.Image, error)
	// Size is the longest side of the thumbnails in pixels, 256 by default.
	Size int
	// Cache is how many thumbnails it keeps, 512 by default.
	Cache int
	// Workers is how many pictures it loads at once, as many as CPUs by
	// default.
	Workers int

	mu sync.Mutex
	// thumbs are the thumbnails made, by key, in lru, the latest shown
	// first.
	thumbs map[K]*list.Element
	lru    list.List
	// queue is the keys waiting, the latest asked for last, and loading
	// those being made.
	queue   []K
	loading map[K]bool
	running int
	wake    func()
}

type thumb[K comparable] struct {
	key K
	bmp *ui.Bitmap
	err error
}

// Thumb returns the thumbnail of key, or the error loading its picture,
// or nil and nil while it is made: it starts making it then, before those
// asked for earlier, and asks c for a frame once made. Call it for the
// thumbnails that show.
func (l *Loader[K]) Thumb(c *ui.Context, key K) (*ui.Bitmap, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.thumbs == nil {
		l.thumbs, l.loading, l.wake = map[K]*list.Element{}, map[K]bool{}, c.Invalidate
		l.Size, l.Cache, l.Workers = or(l.Size, 256), or(l.Cache, 512), or(l.Workers, runtime.NumCPU())
	}
	if e, ok := l.thumbs[key]; ok {
		l.lru.MoveToFront(e)
		t := e.Value.(*thumb[K])
		return t.bmp, t.err
	}
	if l.loading[key] {
		return nil, nil
	}
	if i := slices.Index(l.queue, key); i >= 0 {
		l.queue = slices.Delete(l.queue, i, i+1)
	}
	l.queue = append(l.queue, key)
	if len(l.queue) > maxQueue {
		l.queue = slices.Delete(l.queue, 0, len(l.queue)-maxQueue)
	}
	if l.running < l.Workers {
		l.running++
		go l.work()
	}
	return nil, nil
}

func or(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

// work makes the thumbnails waiting, the latest asked for first, until
// none waits.
func (l *Loader[K]) work() {
	for {
		l.mu.Lock()
		if len(l.queue) == 0 {
			l.running--
			l.mu.Unlock()
			return
		}
		key := l.queue[len(l.queue)-1]
		l.queue = l.queue[:len(l.queue)-1]
		l.loading[key] = true
		size := l.Size
		l.mu.Unlock()

		t := &thumb[K]{key: key}
		img, err := l.Load(key)
		if t.err = err; err == nil {
			t.bmp = ui.NewBitmap(shrink(img, size))
		}

		l.mu.Lock()
		delete(l.loading, key)
		l.thumbs[key] = l.lru.PushFront(t)
		for l.lru.Len() > l.Cache {
			delete(l.thumbs, l.lru.Remove(l.lru.Back()).(*thumb[K]).key)
		}
		wake := l.wake
		l.mu.Unlock()
		wake()
	}
}

// shrink returns img no larger than size pixels either way.
func shrink(img image.Image, size int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= size && h <= size {
		return img
	}
	if w >= h {
		w, h = size, max(1, h*size/w)
	} else {
		w, h = max(1, w*size/h), size
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.BiLinear.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	return dst
}

// View creates a grid of the thumbnails of keys from l, of which s keeps
// the place and the choice, as ui.GridView does: as many columns as fit
// size DIPs wide, each a thumbnail size high, in its own proportions, with
// label of its key below it. Give it a size, or grow it in its parent.
func View[K comparable](c *ui.Context, s *ui.GridState, l *Loader[K], keys []K, size float32, label func(K) string) ui.Element {
	t := c.Theme()
	caption := t.Space(7)
	grid := ui.GridView(c, s, len(keys), size, size+caption, func(i int) {
		key := keys[i]
		bmp, err := l.Thumb(c, key)
		chosen := s.Selected != nil && *s.Selected == i
		ui.Column(c).Grow(1).Gap(t.Space(1)).Children(func() {
			box := ui.Box(c).Grow(1).Radius(t.Space(1.5)).Padding(t.Space(1)).Justify(ui.Center).AlignItems(ui.Center)
			box.Children(func() {
				switch {
				case bmp != nil:
					ui.Image(c, bmp).Fit(ui.Contain).Fill().Radius(t.Space(0.5))
				case err != nil:
					ui.Text(c, "Can't open").FontSize(t.FontSize - 2).TextColor(t.TextMuted)
				default:
					ui.Box(c).Fill().Radius(t.Space(0.5)).Background(t.Surface)
				}
			})
			ui.Row(c).Justify(ui.Center).Children(func() {
				text := ui.Text(c, label(key)).SingleLine().Shrink(1).FontSize(t.FontSize-1).
					Padding(t.Space(0.5), t.Space(1.5)).Radius(t.Space(1))
				if chosen {
					text.Background(t.Accent).TextColor(t.AccentText)
				}
			})
		})
	})
	return grid.Gap(t.Space(2)).Padding(t.Space(3))
}
