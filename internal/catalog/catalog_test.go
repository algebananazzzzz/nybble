package catalog

import "testing"

func TestAddDedupesAndPreservesOrder(t *testing.T) {
	c := Catalog{"Jollibee - Chickenjoy"}
	c = c.Add([]string{"Gyushi - Mala Gyudon", "Jollibee - Chickenjoy"})
	if len(c) != 2 {
		t.Fatalf("want 2 unique, got %d: %v", len(c), c)
	}
	if c[0] != "Jollibee - Chickenjoy" {
		t.Fatal("existing entries should stay first")
	}
}
