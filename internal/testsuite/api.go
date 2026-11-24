package testsuite

import (
	"context"
	"sync"

	"github.com/dembygenesis/spine/internal/api"
	"github.com/dembygenesis/spine/pkg/auth"
	"github.com/dembygenesis/spine/pkg/observe"
)

// SetupAPI builds a Mux for this test: its TestLogger, its FakeAuther, and the
// real tenancy and permission checks reading this test's database. Only the
// token is faked; who may do what is decided exactly as in production.
func (h *Helper) SetupAPI() *api.Mux {
	h.T.Helper()
	db := h.BackendAppDb()

	access, err := api.NewStorageAccess(db)
	if err != nil {
		h.T.Fatalf("access: %v", err)
	}
	catalog, err := auth.LoadCatalog(context.Background(), db)
	if err != nil {
		h.T.Fatalf("catalog: %v", err)
	}
	mux, err := api.New(api.NewParams{
		Logger:  h.Fakers.TestLogger,
		Auther:  h.Fakers.FakeAuther,
		Access:  access,
		Catalog: catalog,
	})
	if err != nil {
		h.T.Fatalf("api.New: %v", err)
	}
	return mux
}

// Fakers are the test doubles a Helper carries.
type Fakers struct {
	TestLogger *observe.TestLogger
	FakeAuther *FakeAuther
}

func newFakers() *Fakers {
	return &Fakers{
		TestLogger: observe.NewTestLogger(),
		FakeAuther: &FakeAuther{},
	}
}

// FakeAuther accepts any non-empty Authorization header as the user set with
// SetUser. A request without the header is a 401, so the unauthenticated case
// needs no token juggling.
type FakeAuther struct {
	mu     sync.Mutex
	userID string
}

// SetUser makes every authenticated request come from userID.
func (f *FakeAuther) SetUser(userID string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.userID = userID
}

func (f *FakeAuther) Authenticate(bearer string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if bearer == "" || f.userID == "" {
		return "", auth.ErrInvalidToken
	}
	return f.userID, nil
}
