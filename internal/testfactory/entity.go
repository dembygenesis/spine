// Package testfactory builds valid rows for tests, so no test hand-writes an
// INSERT or has to know which parent rows a table needs.
package testfactory

import (
	"github.com/dembygenesis/spine/internal/testsuite"

	sq "github.com/Masterminds/squirrel"
)

var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

// Instantiator is the contract every factory meets. The steps are fixed and
// run in this order by New and NewWith, so every factory is used the same way
// and none can skip resolving its foreign keys.
//
// Value receivers that return the factory keep each step a plain
// transformation: f = f.EnsureFKDeps() reads like the pipeline it is.
type Instantiator[T any] interface {
	New(th *testsuite.Helper) T
	EnsureFKDeps() T
	SetRequiredFields() T
	Save() T
	IsValid() bool
}

// New builds and saves a row with every field defaulted:
//
//	org := testfactory.New[testfactory.OrgFactory](th)
func New[F Instantiator[F]](th *testsuite.Helper) F {
	var f F
	return run(th, f)
}

// NewWith builds and saves a row starting from the fields the test cares about;
// everything left empty is defaulted:
//
//	testfactory.NewWith(th, testfactory.OrgFactory{Subject: &model.Organization{Name: "acme"}})
func NewWith[F Instantiator[F]](th *testsuite.Helper, override F) F {
	return run(th, override)
}

func run[F Instantiator[F]](th *testsuite.Helper, f F) F {
	f = f.New(th)
	f = f.EnsureFKDeps()
	f = f.SetRequiredFields()
	return f.Save()
}
