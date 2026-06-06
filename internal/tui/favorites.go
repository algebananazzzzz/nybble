package tui

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/algebananazzzzz/bytecanteen/internal/config"
	"github.com/algebananazzzzz/bytecanteen/internal/menu"
	"github.com/algebananazzzzz/bytecanteen/internal/run"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// favState is shared by pointer between the model and its list delegate, so the
// "grabbed for reorder" flag survives the value-copies Bubble Tea makes. No
// package-level globals.
type favState struct{ grabbed bool }

type favItem string

func (f favItem) FilterValue() string { return string(f) }

// rescanDoneMsg carries the freshly scraped dish catalog and vendor list (or an
// error) back from an on-demand menu rescan.
type rescanDoneMsg struct {
	cat     []string
	vendors []string
	err     error
}

// favDelegate renders one compact, highlightable row per dish.
type favDelegate struct{ st *favState }

func (favDelegate) Height() int                             { return 1 }
func (favDelegate) Spacing() int                            { return 0 }
func (favDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }
func (d favDelegate) Render(w io.Writer, m list.Model, index int, it list.Item) {
	width := m.Width()
	line := fmt.Sprintf("%2d. %s", index+1, string(it.(favItem)))
	switch {
	case index == m.Index() && d.st.grabbed:
		fmt.Fprint(w, grabRowStyle.Width(width).Render(truncate(" "+barGlyph+" "+line, width)))
	default:
		fmt.Fprint(w, row(line, width, index == m.Index()))
	}
}

// favView selects which of the three lists the screen shows. tab cycles
// vendors → dishes → deleted → vendors.
type favView int

const (
	viewVendors favView = iota
	viewDishes
	viewDeleted
)

// favModel is the Favorites & menu screen. It holds two reorderable ranked lists
// — vendors (the fallback ranking, vendors.json) and dishes (favorites.json) —
// plus a deleted-dishes view for restoring items off the persistent exclude list.
type favModel struct {
	vlist       list.Model // ranked vendors (vendors.json)
	list        list.Model // ranked dishes (favorites.json); holds deleted items while view==viewDeleted
	st          *favState  // shared grab flag for the active list
	sp          spinner.Model
	excluded    config.Favorites // persisted blocklist (excluded.json)
	exSet       map[string]bool
	deleted     []deletedDish // in-session quick-undo stack, most recent last
	view        favView
	activeStash []list.Item // dish order parked while the deleted view is shown
	scanning    bool
	notice      string // transient status (saved / deleted / scan result)
}

// deletedDish remembers a delete so it can be undone (reinserted + un-excluded).
type deletedDish struct {
	name  string
	index int
}

// newRankList builds a bare, reorderable list with the shared dish delegate.
func newRankList(items []list.Item, st *favState) list.Model {
	l := list.New(items, favDelegate{st: st}, 40, 10)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(false)
	l.DisableQuitKeybindings()
	return l
}

func newFavModel() *favModel {
	st := &favState{}
	dir, _ := config.ConfigDir()
	favs, _ := config.LoadFavorites(filepath.Join(dir, "favorites.json"))
	cat, _ := config.LoadFavorites(filepath.Join(dir, "catalog.json"))
	excluded, _ := config.LoadFavorites(filepath.Join(dir, "excluded.json"))
	vendors, _ := config.LoadFavorites(filepath.Join(dir, "vendors.json"))

	exSet := map[string]bool{}
	for _, n := range excluded {
		exSet[n] = true
	}

	seen := map[string]bool{}
	var order []list.Item
	add := func(name string) {
		if name != "" && !seen[name] && !exSet[name] {
			order = append(order, favItem(name))
			seen[name] = true
		}
	}
	for _, f := range favs { // ranked favorites first
		add(f)
	}
	for _, c := range cat { // then the rest of the catalog
		add(c)
	}

	// Vendors view: the user's saved ranking first, then any other vendors derived
	// from favorites and the catalog (dishes are named "Vendor - Dish"), so the
	// list is populated from data already on disk without needing a rescan.
	vseen := map[string]bool{}
	var vorder []list.Item
	addV := func(v string) {
		if v != "" && !vseen[v] {
			vorder = append(vorder, favItem(v))
			vseen[v] = true
		}
	}
	for _, v := range vendors { // saved ranking
		addV(v)
	}
	for _, f := range favs { // vendors behind favorite dishes
		addV(menu.Vendor(f))
	}
	for _, c := range cat { // then the rest of the catalog's vendors
		addV(menu.Vendor(c))
	}

	sp := spinner.New(spinner.WithSpinner(spinner.Dot))
	sp.Style = lipgloss.NewStyle().Foreground(cAccent)

	return &favModel{
		vlist:    newRankList(vorder, st),
		list:     newRankList(order, st),
		st:       st,
		sp:       sp,
		excluded: excluded,
		exSet:    exSet,
		view:     viewVendors,
	}
}

// curList returns the list the reorder/scroll keys act on for the current view.
// In the deleted view it is still the dish list (now holding deleted rows).
func (f *favModel) curList() *list.Model {
	if f.view == viewVendors {
		return &f.vlist
	}
	return &f.list
}

func (f *favModel) Update(msg tea.Msg) (screenModel, tea.Cmd) {
	switch msg := msg.(type) {
	case bodySizeMsg:
		h := msg.h - 2 // title + subtitle lines
		if h < 1 {
			h = 1
		}
		f.vlist.SetSize(msg.w, h)
		f.list.SetSize(msg.w, h)
		return f, nil

	case spinner.TickMsg:
		if !f.scanning {
			return f, nil
		}
		var cmd tea.Cmd
		f.sp, cmd = f.sp.Update(msg)
		return f, cmd

	case rescanDoneMsg:
		f.scanning = false
		if msg.err != nil {
			f.notice = errNote("scan failed: " + truncate(msg.err.Error(), 32))
			return f, nil
		}
		added := f.merge(msg.cat)
		vadded := f.mergeVendors(msg.vendors)
		f.notice = okNote(scanNote(added, vadded))
		return f, nil

	case tea.KeyMsg:
		switch f.view {
		case viewDeleted:
			switch msg.String() {
			case "esc", "q":
				return f, nav(scrDashboard)
			case "tab":
				f.cycle()
				return f, nil
			case "d", "x":
				f.restore()
				return f, nil
			case "up", "k", "down", "j", "pgup", "pgdown", "home", "end":
				var cmd tea.Cmd
				f.list, cmd = f.list.Update(msg)
				return f, cmd
			}
			return f, nil

		case viewVendors:
			switch msg.String() {
			case "esc":
				if f.st.grabbed {
					f.st.grabbed = false
					return f, nil
				}
				return f, nav(scrDashboard)
			case "q":
				f.st.grabbed = false
				return f, nav(scrDashboard)
			case "tab":
				if !f.st.grabbed {
					f.cycle()
				}
				return f, nil
			case "s":
				if err := f.saveVendors(); err != nil {
					f.notice = errNote("save failed")
				} else {
					f.notice = okNote("saved")
				}
				return f, nil
			case "enter", " ":
				f.st.grabbed = !f.st.grabbed
				return f, nil
			case "up", "k":
				if f.st.grabbed {
					f.move(-1)
					f.notice = ""
					return f, nil
				}
			case "down", "j":
				if f.st.grabbed {
					f.move(1)
					f.notice = ""
					return f, nil
				}
			case "J", "shift+down":
				f.move(1)
				f.notice = ""
				return f, nil
			case "K", "shift+up":
				f.move(-1)
				f.notice = ""
				return f, nil
			}

		default: // viewDishes
			switch msg.String() {
			case "esc":
				if f.st.grabbed {
					f.st.grabbed = false
					return f, nil
				}
				return f, nav(scrDashboard)
			case "q":
				f.st.grabbed = false
				return f, nav(scrDashboard)
			case "tab":
				if !f.st.grabbed && !f.scanning {
					f.cycle()
				}
				return f, nil
			case "s":
				if err := f.save(); err != nil {
					f.notice = errNote("save failed")
				} else {
					f.notice = okNote("saved")
				}
				return f, nil
			case "d", "x":
				if !f.st.grabbed {
					f.delete()
				}
				return f, nil
			case "u":
				if !f.st.grabbed && !f.scanning {
					f.undoDelete()
				}
				return f, nil
			case "r":
				if !f.scanning && !f.st.grabbed {
					f.scanning = true
					f.notice = ""
					return f, tea.Batch(rescanCmd(), f.sp.Tick)
				}
				return f, nil
			case "enter", " ":
				f.st.grabbed = !f.st.grabbed
				return f, nil
			case "up", "k":
				if f.st.grabbed {
					f.move(-1)
					f.notice = ""
					return f, nil
				}
			case "down", "j":
				if f.st.grabbed {
					f.move(1)
					f.notice = ""
					return f, nil
				}
			case "J", "shift+down":
				f.move(1)
				f.notice = ""
				return f, nil
			case "K", "shift+up":
				f.move(-1)
				f.notice = ""
				return f, nil
			}
		}
	}
	if f.st.grabbed {
		return f, nil // hold focus on the grabbed row
	}
	var cmd tea.Cmd
	cl := f.curList()
	*cl, cmd = cl.Update(msg)
	return f, cmd
}

// cycle advances to the next view: vendors → dishes → deleted → vendors. Entering
// the deleted view parks the dish order; leaving it restores the dish order before
// showing vendors again.
func (f *favModel) cycle() {
	f.notice = ""
	f.st.grabbed = false
	switch f.view {
	case viewVendors:
		f.view = viewDishes
	case viewDishes:
		f.activeStash = f.list.Items()
		f.list.SetItems(f.deletedItems())
		f.list.Select(0)
		f.view = viewDeleted
	case viewDeleted:
		f.list.SetItems(f.activeStash)
		f.activeStash = nil
		f.list.Select(0)
		f.vlist.Select(0)
		f.view = viewVendors
	}
}

// move swaps the selected row with its neighbor and follows it, in whichever
// list the current view owns.
func (f *favModel) move(dir int) {
	l := f.curList()
	items := l.Items()
	i := l.Index()
	j := i + dir
	if j < 0 || j >= len(items) {
		return
	}
	items[i], items[j] = items[j], items[i]
	l.SetItems(items)
	l.Select(j)
}

// delete removes the selected dish, adds it to the persistent exclude list so
// scans never re-add it, and rewrites favorites.json without it — all at once,
// so the deletion sticks even without a manual save.
func (f *favModel) delete() {
	items := f.list.Items()
	i := f.list.Index()
	if i < 0 || i >= len(items) {
		return
	}
	name := string(items[i].(favItem))
	items = append(items[:i:i], items[i+1:]...)
	f.list.SetItems(items)
	if i >= len(items) && len(items) > 0 {
		f.list.Select(len(items) - 1)
	}

	if !f.exSet[name] {
		f.excluded = append(f.excluded, name)
		f.exSet[name] = true
	}
	f.deleted = append(f.deleted, deletedDish{name: name, index: i})
	if err := f.persist(); err != nil { // excluded.json + favorites.json
		f.notice = errNote("delete not saved")
		return
	}
	f.notice = okNote("deleted "+truncate(name, 24)) + footStyle.Render("   ·   u to undo")
}

// undoDelete reverses the most recent delete: the dish is reinserted at its old
// position, dropped from the exclude list, and both files are rewritten.
func (f *favModel) undoDelete() {
	if len(f.deleted) == 0 {
		f.notice = footStyle.Render("nothing to undo")
		return
	}
	last := f.deleted[len(f.deleted)-1]
	f.deleted = f.deleted[:len(f.deleted)-1]

	delete(f.exSet, last.name)
	kept := f.excluded[:0]
	for _, n := range f.excluded {
		if n != last.name {
			kept = append(kept, n)
		}
	}
	f.excluded = kept

	items := f.list.Items()
	idx := last.index
	if idx > len(items) {
		idx = len(items)
	}
	items = append(items, favItem("")) // grow by one
	copy(items[idx+1:], items[idx:])
	items[idx] = favItem(last.name)
	f.list.SetItems(items)
	f.list.Select(idx)

	if err := f.persist(); err != nil {
		f.notice = errNote("undo not saved")
		return
	}
	f.notice = okNote("restored " + truncate(last.name, 24))
}

// deletedItems builds the deleted view from the persisted exclude list.
func (f *favModel) deletedItems() []list.Item {
	items := make([]list.Item, 0, len(f.excluded))
	for _, n := range f.excluded {
		items = append(items, favItem(n))
	}
	return items
}

// restore (deleted view) brings the selected dish back: it drops from the
// exclude list, lands at the bottom of the active order, and both files are
// rewritten. Any matching in-session undo entry is purged so a later u can't
// reinsert a duplicate.
func (f *favModel) restore() {
	items := f.list.Items()
	i := f.list.Index()
	if i < 0 || i >= len(items) {
		return
	}
	name := string(items[i].(favItem))
	items = append(items[:i:i], items[i+1:]...)
	f.list.SetItems(items)
	if i >= len(items) && len(items) > 0 {
		f.list.Select(len(items) - 1)
	}

	delete(f.exSet, name)
	keptEx := f.excluded[:0]
	for _, n := range f.excluded {
		if n != name {
			keptEx = append(keptEx, n)
		}
	}
	f.excluded = keptEx

	keptDel := f.deleted[:0]
	for _, d := range f.deleted {
		if d.name != name {
			keptDel = append(keptDel, d)
		}
	}
	f.deleted = keptDel

	f.activeStash = append(f.activeStash, favItem(name))

	if err := f.persist(); err != nil {
		f.notice = errNote("restore not saved")
		return
	}
	f.notice = okNote("restored " + truncate(name, 24))
}

// activeOrder returns the dish order regardless of which view is on screen, so
// dish saves stay correct while the deleted view (which parks it) is open.
func (f *favModel) activeOrder() []list.Item {
	if f.view == viewDeleted {
		return f.activeStash
	}
	return f.list.Items()
}

// merge appends newly scanned dishes (not already listed and not excluded) to
// the bottom of the dish list, preserving the user's current order. Returns the
// count added.
func (f *favModel) merge(cat []string) int {
	have := map[string]bool{}
	items := f.list.Items()
	for _, it := range items {
		have[string(it.(favItem))] = true
	}
	added := 0
	for _, n := range cat {
		if n == "" || have[n] || f.exSet[n] {
			continue
		}
		items = append(items, favItem(n))
		have[n] = true
		added++
	}
	if added > 0 {
		f.list.SetItems(items)
	}
	return added
}

// mergeVendors appends newly scanned vendors to the bottom of the vendor list,
// preserving the user's ranking. Returns the count added.
func (f *favModel) mergeVendors(vendors []string) int {
	have := map[string]bool{}
	items := f.vlist.Items()
	for _, it := range items {
		have[string(it.(favItem))] = true
	}
	added := 0
	for _, n := range vendors {
		if n == "" || have[n] {
			continue
		}
		items = append(items, favItem(n))
		have[n] = true
		added++
	}
	if added > 0 {
		f.vlist.SetItems(items)
	}
	return added
}

// save writes the current dish order to favorites.json.
func (f *favModel) save() error {
	active := f.activeOrder()
	order := make(config.Favorites, 0, len(active))
	for _, it := range active {
		order = append(order, string(it.(favItem)))
	}
	dir, _ := config.ConfigDir()
	return config.SaveFavorites(filepath.Join(dir, "favorites.json"), order)
}

// saveVendors writes the current vendor order to vendors.json.
func (f *favModel) saveVendors() error {
	order := make(config.Favorites, 0, len(f.vlist.Items()))
	for _, it := range f.vlist.Items() {
		order = append(order, string(it.(favItem)))
	}
	dir, _ := config.ConfigDir()
	return config.SaveFavorites(filepath.Join(dir, "vendors.json"), order)
}

// persist writes both the exclude list and the dish order, used after a delete
// or undo. Returns the first write error.
func (f *favModel) persist() error {
	dir, _ := config.ConfigDir()
	if err := config.SaveFavorites(filepath.Join(dir, "excluded.json"), f.excluded); err != nil {
		return err
	}
	return f.save()
}

func (f *favModel) View(w, h int) string {
	switch f.view {
	case viewVendors:
		return f.vendorsView(w)
	case viewDeleted:
		return f.deletedView(w)
	default:
		return f.dishesView(w)
	}
}

func (f *favModel) vendorsView(w int) string {
	title := titleStyle.Render("Vendors")
	if f.st.grabbed {
		title += warnStyle.Render("   moving")
	}
	head := title
	if f.notice != "" && !f.st.grabbed {
		head = spread(title, f.notice, w)
	}
	var b strings.Builder
	b.WriteString(head + "\n")
	b.WriteString(subtitleStyle.Render("rank vendors — fallback when no favorite dish is in stock") + "\n")
	if len(f.vlist.Items()) == 0 {
		b.WriteString(footStyle.Render("no vendors yet — tab to dishes and press r to rescan"))
		return b.String()
	}
	b.WriteString(f.vlist.View())
	return b.String()
}

func (f *favModel) dishesView(w int) string {
	title := titleStyle.Render("Dishes")
	if f.st.grabbed {
		title += warnStyle.Render("   moving")
	}
	// Transient status rides the title row (right-aligned) so the footer hints
	// always stay complete.
	head := title
	if f.notice != "" && !f.st.grabbed {
		head = spread(title, f.notice, w)
	}
	var b strings.Builder
	b.WriteString(head + "\n")
	b.WriteString(subtitleStyle.Render("rank dishes — top of the list books first") + "\n")
	b.WriteString(f.list.View())
	return b.String()
}

func (f *favModel) deletedView(w int) string {
	title := titleStyle.Render("Deleted dishes")
	head := title
	if f.notice != "" {
		head = spread(title, f.notice, w)
	}
	var b strings.Builder
	b.WriteString(head + "\n")
	b.WriteString(subtitleStyle.Render("restore dishes back to the menu") + "\n")
	if len(f.list.Items()) == 0 {
		b.WriteString(footStyle.Render("no deleted dishes"))
		return b.String()
	}
	b.WriteString(f.list.View())
	return b.String()
}

func (f *favModel) Footer() string {
	if f.scanning {
		return f.sp.View() + footStyle.Render(" scanning menu…   esc back")
	}
	switch f.view {
	case viewVendors:
		if f.st.grabbed {
			return footStyle.Render("↑/↓ move vendor   enter drop   s save   esc cancel")
		}
		return footStyle.Render("↑/↓ scroll   enter grab   tab dishes   s save   esc back")
	case viewDeleted:
		return footStyle.Render("↑/↓ scroll   d restore   tab vendors   esc back")
	default: // dishes
		if f.st.grabbed {
			return footStyle.Render("↑/↓ move dish   enter drop   s save   esc cancel")
		}
		hint := "↑/↓ scroll  enter grab  d delete"
		if len(f.deleted) > 0 {
			hint += "  u undo"
		}
		hint += "  r rescan  tab deleted  s save  esc back"
		return footStyle.Render(hint)
	}
}

// scanNote summarizes what a rescan added across dishes and vendors.
func scanNote(dishes, vendors int) string {
	if dishes == 0 && vendors == 0 {
		return "scan: up to date"
	}
	var parts []string
	if dishes > 0 {
		parts = append(parts, fmt.Sprintf("+%d dishes", dishes))
	}
	if vendors > 0 {
		parts = append(parts, fmt.Sprintf("+%d vendors", vendors))
	}
	return "scan: " + strings.Join(parts, " ")
}

// rescanCmd scrapes the upcoming-week menu (network) off the event loop and
// reports the updated dish catalog and vendor list back as a rescanDoneMsg.
func rescanCmd() tea.Cmd {
	return func() tea.Msg {
		d, err := run.LoadDeps()
		if err != nil {
			return rescanDoneMsg{err: err}
		}
		cat, vendors, err := run.Scan(d)
		return rescanDoneMsg{cat: []string(cat), vendors: []string(vendors), err: err}
	}
}
