package postgres

import (
	"context"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pbabbicola/go-monitor/monitor"
)

func Test_writeToPostgres(t *testing.T) {
	timestamp, err := time.Parse(time.RFC3339, "2021-10-14T16:08:01+01:00")
	require.NoError(t, err)

	logger := slog.Default()
	defer slog.SetDefault(logger)

	slog.SetDefault(slog.New(slog.DiscardHandler))

	tests := []struct {
		name           string
		batch          []monitor.Message
		wantErr        bool
		dbExpectations func(mock sqlmock.Sqlmock)
	}{
		{
			name: "happy path",
			batch: []monitor.Message{
				{
					URL:           "some_url",
					Duration:      time.Second,
					Timestamp:     timestamp,
					StatusCode:    http.StatusOK,
					RegexpMatches: true,
					Err:           assert.AnError,
				},
			},
			wantErr: false,
			dbExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectPrepare("insert into logs \\(ts, url, duration_milliseconds, status_code, regexp_matches, error\\) values\\(\\$1, \\$2, \\$3, \\$4, \\$5, \\$6\\)").ExpectExec().WithArgs(timestamp, "some_url", int(time.Second/time.Millisecond), http.StatusOK, true, assert.AnError.Error()).WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
		},
		{
			name:    "failed to begin",
			batch:   []monitor.Message{},
			wantErr: true,
			dbExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(assert.AnError)
			},
		},
		{
			name:    "failed to prepare",
			batch:   []monitor.Message{},
			wantErr: true,
			dbExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectPrepare("insert into logs \\(ts, url, duration_milliseconds, status_code, regexp_matches, error\\) values\\(\\$1, \\$2, \\$3, \\$4, \\$5, \\$6\\)").WillReturnError(assert.AnError)
			},
		},
		{
			name: "failed to execute, but error is ignored for the following insert",
			batch: []monitor.Message{
				{
					URL:           "some_url",
					Duration:      time.Second,
					Timestamp:     timestamp,
					StatusCode:    http.StatusOK,
					RegexpMatches: true,
					Err:           assert.AnError,
				},
				{
					URL:           "some_url_2",
					Duration:      2 * time.Second,
					Timestamp:     timestamp.Add(time.Hour),
					StatusCode:    http.StatusNotAcceptable,
					RegexpMatches: false,
					Err:           nil,
				},
			},
			wantErr: false,
			dbExpectations: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()

				prepared := mock.ExpectPrepare("insert into logs \\(ts, url, duration_milliseconds, status_code, regexp_matches, error\\) values\\(\\$1, \\$2, \\$3, \\$4, \\$5, \\$6\\)")

				prepared.ExpectExec().
					WithArgs(timestamp, "some_url", int(time.Second/time.Millisecond), http.StatusOK, true, assert.AnError.Error()).
					WillReturnError(err)

				prepared.ExpectExec().
					WithArgs(timestamp.Add(time.Hour), "some_url_2", int(2*time.Second/time.Millisecond), http.StatusNotAcceptable, false, "").
					WillReturnResult(sqlmock.NewResult(1, 1))

				mock.ExpectCommit()
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, mock, err := sqlmock.New()
			require.NoError(t, err)

			tt.dbExpectations(mock)

			err = writeToPostgres(context.Background(), pool, tt.batch)
			assert.Truef(t, err != nil == tt.wantErr, "error was %v and wantErr was %v", err, tt.wantErr)
			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}
