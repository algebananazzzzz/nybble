package tui

import (
	"path/filepath"
	"testing"

	"github.com/algebananazzzzz/bytecanteen/internal/config"
	"github.com/charmbracelet/bubbles/list"
)

func newTestFav(names ...string) *favModel {
	st := &favState{}
	items := make([]list.Item, len(names))
	for i, n := range names {
		items[i] = favItem(n)
	}
	l := list.New(items, favDelegate{st: st}, 40, 10)
	l.SetFilteringEnabled(false)
	vl := list.New(nil, favDelegate{st: st}, 40, 10)
	vl.SetFilteringEnabled(false)
	return &favModel{list: l, vlist: vl, st: st, exSet: map[string]bool{}, view: viewDishes}
}

func itemNames(f *favModel) []string {
	out := []string{}
	for _, it := range f.list.Items() {
		out = append(out, string(it.(favItem)))
	}
	return out
}

func TestDeleteRemovesAndExcludes(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // isolate disk writes
	f := newTestFav("a", "b", "c")
	f.list.Select(1) // b

	f.delete()

	got := itemNames(f)
	if len(got) != 2 || got[0] != "a" || got[1] != "c" {
		t.Fatalf("after delete got %v, want [a c]", got)
	}
	if !f.exSet["b"] {
		t.Fatal("deleted dish b should be in the exclude set")
	}
}

func TestUndoRestoresAtOriginalIndex(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	f := newTestFav("a", "b", "c")
	f.list.Select(1) // b
	f.delete()
	if itemNames(f)[0] != "a" || len(itemNames(f)) != 2 || !f.exSet["b"] {
		t.Fatalf("delete precondition failed: %v", itemNames(f))
	}

	f.undoDelete()

	got := itemNames(f)
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "c" {
		t.Fatalf("after undo got %v, want [a b c]", got)
	}
	if f.exSet["b"] {
		t.Fatal("undone dish b should no longer be excluded")
	}
	if len(f.deleted) != 0 {
		t.Fatalf("undo stack should be empty, got %d", len(f.deleted))
	}
}

func TestCycleVisitsAllThreeViews(t *testing.T) {
	f := newTestFav("a", "b")
	f.excluded = append(f.excluded, "x", "y")
	f.exSet["x"], f.exSet["y"] = true, true
	f.view = viewVendors // default landing view

	f.cycle() // vendors -> dishes
	if f.view != viewDishes {
		t.Fatalf("after first cycle view = %d, want dishes", f.view)
	}

	f.cycle() // dishes -> deleted
	if f.view != viewDeleted {
		t.Fatalf("after second cycle view = %d, want deleted", f.view)
	}
	got := itemNames(f)
	if len(got) != 2 || got[0] != "x" || got[1] != "y" {
		t.Fatalf("deleted view %v, want [x y]", got)
	}

	f.cycle() // deleted -> vendors (dish order restored to f.list)
	if f.view != viewVendors {
		t.Fatalf("after third cycle view = %d, want vendors", f.view)
	}
	got = itemNames(f) // f.list is the dish list again
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("dish list after full cycle %v, want [a b]", got)
	}
}

func TestRestoreMovesDishToActiveBottom(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir()) // isolate disk writes
	f := newTestFav("a", "b")
	f.excluded = append(f.excluded, "x", "y")
	f.exSet["x"], f.exSet["y"] = true, true

	f.cycle() // dishes -> deleted view: [x y]
	f.list.Select(0)
	f.restore() // restore x

	got := itemNames(f)
	if len(got) != 1 || got[0] != "y" {
		t.Fatalf("deleted view after restore %v, want [y]", got)
	}
	if f.exSet["x"] {
		t.Fatal("restored dish x should no longer be excluded")
	}
	if len(f.excluded) != 1 || f.excluded[0] != "y" {
		t.Fatalf("excluded list %v, want [y]", f.excluded)
	}

	f.cycle() // deleted -> vendors: dish list restored with x appended
	got = itemNames(f)
	if len(got) != 3 || got[0] != "a" || got[1] != "b" || got[2] != "x" {
		t.Fatalf("dish list after restore %v, want [a b x]", got)
	}
}

func TestVendorReorderAndSave(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	f := newTestFav() // empty dish list
	f.view = viewVendors
	f.vlist.SetItems([]list.Item{favItem("StarStall"), favItem("Corner")})

	// grab the top vendor, move it down one
	f.st.grabbed = true
	f.move(1)
	f.st.grabbed = false

	if err := f.saveVendors(); err != nil {
		t.Fatalf("saveVendors: %v", err)
	}
	dir, _ := config.ConfigDir()
	got, _ := config.LoadFavorites(filepath.Join(dir, "vendors.json"))
	if len(got) != 2 || got[0] != "Corner" || got[1] != "StarStall" {
		t.Fatalf("vendors.json = %v, want [Corner StarStall]", got)
	}
}

func TestMergeDedupsAndRespectsExclude(t *testing.T) {
	f := newTestFav("a", "b")
	f.exSet["x"] = true // previously deleted

	added := f.merge([]string{"b", "x", "c", "c", ""})

	if added != 1 {
		t.Fatalf("added = %d, want 1 (only c is new)", added)
	}
	got := itemNames(f)
	if len(got) != 3 || got[2] != "c" {
		t.Fatalf("merged list %v, want [a b c]", got)
	}
}
