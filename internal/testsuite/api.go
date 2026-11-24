package testsuite

import (
	"github.com/dembygenesis/spine/internal/api"
	"github.com/dembygenesis/spine/pkg/observe"
)

// SetupAPI builds a Mux whose logger is this test's TestLogger, so the test can
// assert which errors were logged.
func (h *Helper) SetupAPI() *api.Mux {
	h.T.Helper()
	mux, err := api.New(api.NewParams{Logger: h.Fakers.TestLogger})
	if err != nil {
		h.T.Fatalf("api.New: %v", err)
	}
	return mux
}

// Fakers are the test doubles a Helper carries.
type Fakers struct {
	TestLogger *observe.TestLogger
}

func newFakers() *Fakers {
	return &Fakers{TestLogger: observe.NewTestLogger()}
}
