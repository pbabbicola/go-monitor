package monitor_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/[REDACTED]-recruiting/go-20250912-pbabbicola/config"
	"github.com/[REDACTED]-recruiting/go-20250912-pbabbicola/monitor"
)

type mockedMonitorer struct {
	monitored int
	err       error
}

func NewMockedMonitorer(err error) *mockedMonitorer {
	return &mockedMonitorer{
		monitored: 0,
		err:       err,
	}
}

func (m *mockedMonitorer) monitor(context.Context, config.SiteElement) error {
	m.monitored++

	return m.err
}

func TestTicks(t *testing.T) {
	tests := []struct {
		name string // Description of this test case
		// Named input parameters for target function.
		website   config.SiteElement
		monitorer *mockedMonitorer
		// Other needed parameters
		expectedAmountOfCalls int
		timeout               time.Duration
	}{
		{
			name: "test ticks of random url",
			website: config.SiteElement{
				URL:             "test-url",
				Regexp:          nil,
				IntervalSeconds: 1,
			},
			monitorer:             NewMockedMonitorer(nil),
			expectedAmountOfCalls: 5,
			timeout:               5 * time.Second,
		},
		{
			name: "test ticks that are cancelled before the monitoring has time to run",
			website: config.SiteElement{
				URL:             "test-url",
				Regexp:          nil,
				IntervalSeconds: 5,
			},
			monitorer:             NewMockedMonitorer(nil),
			expectedAmountOfCalls: 0,
			timeout:               1 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
				defer cancel()

				monitor.Ticks(ctx, tt.website, tt.monitorer.monitor)

				assert.Equal(t, tt.expectedAmountOfCalls, tt.monitorer.monitored)
			})
		})
	}
}

type mockedHandler struct {
	statusCode int
}

func NewMockedHandler(statusCode int) *mockedHandler {
	return &mockedHandler{
		statusCode: statusCode,
	}
}

func (s *mockedHandler) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(s.statusCode)
}

// TODO: Test other cases.
func TestDefaultMonitorer_Monitor(t *testing.T) {
	tests := []struct {
		name string // Description of this test case
		// Named input parameters for target function.
		website config.SiteElement
		// Other needed parameters
		wantErr     bool
		fakeHandler *mockedHandler
	}{
		{
			name: "happy path",
			website: config.SiteElement{
				URL: "https://example.org",
			},
			wantErr:     false,
			fakeHandler: NewMockedHandler(http.StatusOK),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// creates a fake server for our fake handler
			fakeServer := httptest.NewServer(tt.fakeHandler)
			defer fakeServer.Close()

			messageQueue := make(chan monitor.Message)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			m := monitor.NewDefaultMonitorer(fakeServer.Client(), messageQueue)

			var wg sync.WaitGroup

			wg.Go(func() {
				err := m.Monitor(ctx, tt.website)

				cancel() // cancels the context so the receiver knows to stop

				assert.Truef(t, err != nil == tt.wantErr, "wanted err to be %v, but got error %v", tt.wantErr, err)
			})

			// Create a receiver/consumer similar to consumers/log, but instead of logging, this checks that we are being sent a the expected website log. Probably it could've done with a few more assert checks on other fields.
			wg.Go(func() {
				for {
					select {
					case <-ctx.Done():
						return
					case msg := <-messageQueue:
						if !tt.wantErr {
							assert.Equal(t, tt.website.URL, msg.URL)
						}

						return
					}
				}
			})

			wg.Wait()
		})
	}
}

func TestDefaultMonitorer_Monitor_Errors(t *testing.T) {
	tests := []struct {
		name      string
		website   config.SiteElement
		wantErr   error
		monitorer *monitor.DefaultMonitorer
	}{
		{
			name: "nil monitorer",
			website: config.SiteElement{
				URL: "https://example.org",
			},
			wantErr:   monitor.ErrNilMonitorer,
			monitorer: nil,
		},
		{
			name: "nil client",
			website: config.SiteElement{
				URL: "https://example.org",
			},
			wantErr:   monitor.ErrNilClient,
			monitorer: &monitor.DefaultMonitorer{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.monitorer.Monitor(context.Background(), tt.website)

			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}
