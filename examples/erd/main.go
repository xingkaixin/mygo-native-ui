// Erd draws the diagram of an online shop's database from its SQL, beside
// the SQL in an editor, and draws it again as the SQL changes, with what
// is wrong in it listed below by line. Drag a table by its name to move
// it; point at a table, or click a column, to light its foreign keys.
//
//	go run ./examples/erd
package main

import (
	"fmt"
	"log"
	"strconv"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"

	"github.com/xingkaixin/mygo-native-ui/ui/erd"
)

const shop = `-- An online shop.

CREATE TABLE users (
  id         bigserial PRIMARY KEY,
  email      varchar(255) NOT NULL UNIQUE,
  name       text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE addresses (
  id          bigserial PRIMARY KEY,
  user_id     bigint NOT NULL
              REFERENCES users (id) ON DELETE CASCADE,
  line1       text NOT NULL,
  line2       text,
  city        text NOT NULL,
  postal_code varchar(16) NOT NULL,
  country     char(2) NOT NULL DEFAULT 'US'
);

CREATE TABLE categories (
  id        serial PRIMARY KEY,
  parent_id integer REFERENCES categories,
  name      text NOT NULL,
  slug      text NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS products (
  id          bigserial PRIMARY KEY,
  category_id integer NOT NULL REFERENCES categories (id),
  sku         varchar(32) NOT NULL UNIQUE,
  name        text NOT NULL,
  price       numeric(10,2) NOT NULL CHECK (price >= 0),
  stock       integer NOT NULL DEFAULT 0
);

CREATE TABLE orders (
  id                  bigserial PRIMARY KEY,
  user_id             bigint NOT NULL,
  shipping_address_id bigint,
  status              varchar(20) NOT NULL DEFAULT 'pending',
  placed_at           timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT orders_user_fk
    FOREIGN KEY (user_id) REFERENCES users (id),
  FOREIGN KEY (shipping_address_id)
    REFERENCES addresses (id) ON DELETE SET NULL
);

CREATE TABLE order_items (
  order_id   bigint NOT NULL
             REFERENCES orders (id) ON DELETE CASCADE,
  product_id bigint NOT NULL REFERENCES products (id),
  quantity   integer NOT NULL CHECK (quantity > 0),
  unit_price numeric(10,2) NOT NULL,
  PRIMARY KEY (order_id, product_id)
);

CREATE TABLE reviews (
  id         bigserial PRIMARY KEY,
  product_id bigint NOT NULL REFERENCES products (id),
  user_id    bigint REFERENCES users (id) ON DELETE SET NULL,
  rating     smallint NOT NULL CHECK (rating BETWEEN 1 AND 5),
  body       text,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE public."payments" (
  id       bigserial PRIMARY KEY,
  order_id bigint NOT NULL REFERENCES orders (id),
  provider varchar(32) NOT NULL,
  amount   numeric(10,2) NOT NULL,
  /* Null until the provider confirms it. */
  paid_at  timestamptz
);

CREATE INDEX reviews_product_idx ON reviews (product_id);
`

type app struct {
	sql     string
	shown   bool
	tables  []erd.Table
	errs    []erd.Error
	diagram erd.State
}

func (a *app) view(c *ui.Context) {
	t := c.Theme()
	ui.Row(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Width(t.Space(135)).FillHeight().Children(func() {
			sql := ui.TextAreaBase(c, &a.sql).Font("monospace").FontSize(t.FontSize-1).
				Padding(t.Space(3), t.Space(4)).Label("SQL").Grow(1)
			if !a.shown {
				// A text area starts with the caret after its text: show the
				// schema from the top.
				sql.SetTextSelection(0, 0)
				a.shown = true
			}
			if sql.Changed() {
				a.tables, a.errs = erd.Parse(a.sql)
			}
			ui.Divider(c)
			ui.Column(c).Gap(t.Space(1.5)).Padding(t.Space(3), t.Space(4)).Children(func() {
				if len(a.errs) == 0 {
					keys := 0
					for _, tb := range a.tables {
						for _, col := range tb.Columns {
							if col.RefTable != "" {
								keys++
							}
						}
					}
					ui.Text(c, fmt.Sprintf("%d tables, %d foreign keys", len(a.tables), keys)).
						FontSize(t.FontSize - 1).TextColor(t.TextMuted)
					return
				}
				for i, e := range a.errs {
					ui.Row(c.Key(i)).Gap(t.Space(2)).Children(func() {
						ui.Text(c, "Line "+strconv.Itoa(e.Line)).FontSize(t.FontSize - 1).FontFeatures("tnum").
							TextColor(t.TextMuted).Shrink(0)
						ui.Text(c, e.Msg).FontSize(t.FontSize - 1).TextColor(t.Danger).Shrink(1)
					})
				}
			})
		})
		ui.Divider(c)
		erd.View(c, &a.diagram, a.tables).Grow(1).FillHeight().Background(t.Surface)
	})
}

func main() {
	a := &app{sql: shop}
	a.tables, a.errs = erd.Parse(a.sql)
	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Schema",
			Width:   1600,
			Height:  900,
			Content: ui.View(a.view),
		})
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
