package allocator

import (
	"fmt"
	"log"
	"time"

	"github.com/irfansharif/solver"
)

type Item int

func (i Item) String() string {
	return fmt.Sprintf("item-%d", i)
}

type Bin int

func (b Bin) String() string {
	return fmt.Sprintf("bin-%d", b)
}

type Resource int

func (r Resource) String() string {
	return fmt.Sprintf("resource-%d", r)
}

type ItemBin struct {
	Item
	Bin
}

type Allocator struct {
	// items placed in bin, each item can consume resources of different costs,
	// each bin has a limited capacity used by items in that bin, per their
	// resource requirements.

	Items     []Item
	Bins      []Bin
	Resources []Resource

	Literals      map[ItemBin]solver.Literal
	LastPlacement map[Item]Bin

	Options struct {
		DisableEvenDistribution bool
		DisableMaxChurn         bool
		DisableCapacityChecking bool
	}
}

func (a *Allocator) Copies(i Item) int {
	return 1 // fixed replication factor
}

func (a *Allocator) TotalCopies(is []Item) int {
	return 1 * len(is) // fixed replication factor
}

func (a *Allocator) Required(i Item, r Resource) int {
	return 1 // fixed resource cost independent of type
}

func (a *Allocator) Capacity(b Bin) int64 {
	return 10 // fixed capacity per bucket
}

func (a *Allocator) MaxChurn() int {
	return 10 // place maximum of ten items per round
}

func (a *Allocator) AddItem() {
	last := a.Items[len(a.Items)-1]
	a.Items = append(a.Items, Item(int(last)+1))
}

func (a *Allocator) DropItem() {
	a.Items = a.Items[:len(a.Items)-1]
}

func NewAllocator(items, bins, resources int) *Allocator { // resources per item
	is := make([]Item, 0, items)
	for i := 0; i < items; i++ {
		is = append(is, Item(i))
	}

	bs := make([]Bin, 0, bins)
	for i := 0; i < bins; i++ {
		bs = append(bs, Bin(i))
	}

	rs := make([]Resource, 0, resources)
	for i := 0; i < resources; i++ {
		rs = append(rs, Resource(i))
	}

	return &Allocator{
		Items:         is,
		Bins:          bs,
		Resources:     rs,
		Literals:      make(map[ItemBin]solver.Literal),
		LastPlacement: make(map[Item]Bin),
	}
}

// TODO(irfansharif): Use intervals for per-node constraints
// https://github.com/google/or-tools/issues/1799

func (a *Allocator) Allocate() (placement map[Item]Bin, ok bool) {
	model := solver.NewModel("allocator")

	// We'll instantiate a literal for every item-bin pair.
	for _, item := range a.Items {
		for _, bin := range a.Bins {
			lit := model.NewLiteral(fmt.Sprintf("%s_in_%s", item, bin))
			a.add(item, bin, lit)
		}
	}

	for _, item := range a.Items {
		// Place each item exactly n times, where n is the number of copies we
		// want to place.
		constraint := solver.NewExactlyKConstraint(a.Copies(item), a.itemliterals(item)...)
		model.AddConstraints(constraint)
	}

	if !a.Options.DisableCapacityChecking { // Ensure placements respect each bin's capacity.
		for _, bin := range a.Bins {
			vars := solver.AsIntVars(a.binliterals(bin))
			placed := solver.Sum(vars...)
			capacity := solver.NewDomain(0, a.Capacity(bin))

			model.AddConstraints(solver.NewLinearConstraint(placed, capacity))
		}
	}

	if !a.Options.DisableEvenDistribution { // Evenly distribute item placements.
		avg := a.TotalCopies(a.Items) / len(a.Bins)
		var surpluses []solver.LinearExpr
		for _, bin := range a.Bins {
			vars := solver.AsIntVars(a.binliterals(bin))

			// Capture the number of items we've placed above the best possible
			// average (this is essentially placed - avg).
			surplus := solver.NewLinearExpr(vars, ones(len(vars)), int64(-avg))
			surpluses = append(surpluses, surplus)
		}

		// Capture the maximum surplus.
		maxsurplus := expr(model.NewIntVar(0, 100, "max-surplus"))
		model.AddConstraints(solver.NewLinearMaximumConstraint(maxsurplus, surpluses...))
		// Minimize the maximum surplus, for even distribution. We're asking the
		// solver to "push down" from above.
		model.Minimize(maxsurplus)
	}

	if !a.Options.DisableMaxChurn { // Limit the number of items that are moved around.
		if len(a.LastPlacement) != 0 {
			var literals []solver.Literal
			for _, bin := range a.Bins {
				for _, item := range a.Items {
					if !a.placed(item, bin) {
						continue
					}

					// This item-bin pair was previously placed. If it's not
					// placed this round (the literal is assigned false), we
					// want to count it -- get its negation.
					literals = append(literals, a.get(item, bin).Not())
				}
			}
			model.AddConstraints(
				solver.NewAtMostKConstraint(a.MaxChurn(), literals...),
			)
		}
	}
	log.Printf("%s", model.String())

	if ok, _ := model.Validate(); !ok {
		return nil, false
	}

	start := time.Now()
	result := model.Solve()
	log.Printf("duration: %s", time.Since(start))
	if !result.Optimal() {
		return nil, false
	}

	placement = make(map[Item]Bin)
	for _, item := range a.Items {
		for _, bin := range a.Bins {
			if !result.BooleanValue(a.get(item, bin)) {
				continue
			}

			placement[item] = bin
		}
	}
	a.LastPlacement = placement
	return placement, true
}

func expr(iv solver.IntVar) solver.LinearExpr {
	return solver.Sum(iv)
}

func ones(l int) []int64 {
	var res []int64
	for i := 0; i < l; i += 1 {
		res = append(res, 1)
	}
	return res
}

func (a *Allocator) add(item Item, bin Bin, l solver.Literal) {
	a.Literals[ItemBin{item, bin}] = l
}

func (a *Allocator) get(item Item, bin Bin) solver.Literal {
	return a.Literals[ItemBin{item, bin}]
}

func (a *Allocator) placed(item Item, bin Bin) bool {
	b, ok := a.LastPlacement[item]
	return ok && b == bin
}

func (a *Allocator) itemliterals(item Item) []solver.Literal {
	var lits []solver.Literal
	for ib, l := range a.Literals {
		if ib.Item == item {
			lits = append(lits, l)
		}
	}
	return lits
}

func (a *Allocator) binliterals(bin Bin) []solver.Literal {
	var lits []solver.Literal
	for ib, l := range a.Literals {
		if ib.Bin == bin {
			lits = append(lits, l)
		}
	}
	return lits
}
