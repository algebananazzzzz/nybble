package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func TestReorderMovesItemDown(t *testing.T) {
	st := &favState{}
	l := list.New([]list.Item{favItem("a"), favItem("b"), favItem("c")}, favDelegate{st: st}, 40, 10)
	l.SetFilteringEnabled(false)
	f := &favModel{list: l, st: st, view: viewDishes}

	sm, _ := f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}})
	f = sm.(*favModel)

	got := f.list.Items()
	if string(got[0].(favItem)) != "b" || string(got[1].(favItem)) != "a" {
		t.Fatalf("reorder failed: got %q,%q", got[0].(favItem), got[1].(favItem))
	}
	if f.list.Index() != 1 {
		t.Fatalf("cursor should follow item to index 1, got %d", f.list.Index())
	}
}
