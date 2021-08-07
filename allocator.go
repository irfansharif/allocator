package allocator

import (
	"fmt"

	"github.com/irfansharif/or-tools/cpsatsolver"
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

	Literals map[ItemBin]cpsatsolver.Literal
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
		Items:     is,
		Bins:      bs,
		Resources: rs,
		Literals:  make(map[ItemBin]cpsatsolver.Literal),
	}
}

// TODO(irfansharif): Use intervals for per-node constraints
// https://github.com/google/or-tools/issues/1799

func (a *Allocator) Allocate() (placement map[Item]Bin, ok bool) {
	model := cpsatsolver.NewModel()

	// We'll instantiate a literal for every item-bin pair.
	for _, item := range a.Items {
		for _, bin := range a.Bins {
			lit := model.NewLiteral(fmt.Sprintf("%s in %s", item, bin))
			a.add(item, bin, lit)
		}
	}

	for _, item := range a.Items {
		// Place each item exactly n times, where n is the number of copies we
		// want to place.
		constraint := cpsatsolver.NewExactlyKConstraint(a.Copies(item), a.itemliterals(item)...)
		model.AddConstraints(constraint)
	}

	avg := a.TotalCopies(a.Items) / len(a.Bins)
	var surpluses []cpsatsolver.LinearExpr
	for _, bin := range a.Bins {
		vars := intvars(a.binliterals(bin))
		placed := cpsatsolver.Sum(vars...)
		capacity := cpsatsolver.NewDomain(0, a.Capacity(bin))

		// Ensure placements respect each bin's capacity.
		model.AddConstraints(cpsatsolver.NewLinearConstraint(placed, capacity))

		// Capture the number of items we've placed above the best possible
		// average (this is essentially placed - avg).
		surplus := cpsatsolver.NewLinearExpr(vars, ones(len(vars)), int64(-avg))
		surpluses = append(surpluses, surplus)
	}

	// Capture the maximum surplus.
	maxsurplus := expr(model.NewIntVar(0, 100, "max-surplus"))
	model.AddConstraints(cpsatsolver.NewLinearMaximumConstraint(maxsurplus, surpluses...))

	// Minimize the maximum surplus, for even distribution. We're asking the
	// solver to "push down" from above.
	model.Minimize(maxsurplus)
	if ok, _ := model.Validate(); !ok {
		return nil, false
	}

	result := model.Solve()
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

	return placement, true
}

func expr(iv cpsatsolver.IntVar) cpsatsolver.LinearExpr {
	return cpsatsolver.Sum(iv)
}

func ones(l int) []int64 {
	var res []int64
	for i := 0; i < l; i += 1 {
		res = append(res, 1)
	}
	return res
}

func (a *Allocator) add(item Item, bin Bin, l cpsatsolver.Literal) {
	a.Literals[ItemBin{item, bin}] = l
}

func (a *Allocator) get(item Item, bin Bin) cpsatsolver.Literal {
	return a.Literals[ItemBin{item, bin}]
}

func (a *Allocator) itemliterals(item Item) []cpsatsolver.Literal {
	var lits []cpsatsolver.Literal
	for ib, l := range a.Literals {
		if ib.Item == item {
			lits = append(lits, l)
		}
	}
	return lits
}

func (a *Allocator) binliterals(bin Bin) []cpsatsolver.Literal {
	var lits []cpsatsolver.Literal
	for ib, l := range a.Literals {
		if ib.Bin == bin {
			lits = append(lits, l)
		}
	}
	return lits
}

func intvars(literals []cpsatsolver.Literal) []cpsatsolver.IntVar {
	var res []cpsatsolver.IntVar
	for _, l := range literals {
		res = append(res, l.(cpsatsolver.IntVar))
	}
	return res
}
