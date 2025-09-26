package batcher

import (
	"context"
	"database/sql"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/pbabbicola/go-monitor/monitor"
)

// This test doesn't need to be a table test, but I find them more readable even if they have one element. I tried doing a non-table test somewhere else in this project but I didn't quite like it.
func TestBatcher_Add(t *testing.T) {
	tests := []struct {
		name          string
		batcher       *Batcher
		expectedBatch []monitor.Message
	}{
		{
			name: "happy path",
			batcher: &Batcher{
				mut:       &sync.Mutex{},
				batch:     []monitor.Message{},
				batchSize: 10,
			},
			expectedBatch: []monitor.Message{
				{StatusCode: 0},
				{StatusCode: 1},
				{StatusCode: 2},
				{StatusCode: 3},
				{StatusCode: 4},
				{StatusCode: 5},
				{StatusCode: 6},
				{StatusCode: 7},
				{StatusCode: 8},
				{StatusCode: 9},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup

			for i := 0; i < tt.batcher.batchSize; i++ {
				wg.Go(func() {
					tt.batcher.Add(monitor.Message{StatusCode: i})
				})
			}

			wg.Wait()

			assert.ElementsMatch(t, tt.expectedBatch, tt.batcher.batch)
		})
	}
}

// This test also doesn't _need_ to be a table test. I think right now doing a good test for this would be a bit cumbersome, so it only checks that the pointers behave as expected.
func TestBatcher_DuplicateAndClear(t *testing.T) {
	tests := []struct {
		name          string
		batcher       *Batcher
		expectedBatch []monitor.Message
	}{
		{
			name: "happy path",
			batcher: &Batcher{
				mut: &sync.Mutex{},
				batch: []monitor.Message{
					{StatusCode: 0},
					{StatusCode: 1},
					{StatusCode: 2},
					{StatusCode: 3},
					{StatusCode: 4},
					{StatusCode: 5},
					{StatusCode: 6},
					{StatusCode: 7},
					{StatusCode: 8},
					{StatusCode: 9},
				},
				batchSize: 10,
			},
			expectedBatch: []monitor.Message{
				{StatusCode: 0},
				{StatusCode: 1},
				{StatusCode: 2},
				{StatusCode: 3},
				{StatusCode: 4},
				{StatusCode: 5},
				{StatusCode: 6},
				{StatusCode: 7},
				{StatusCode: 8},
				{StatusCode: 9},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			duplicated := tt.batcher.DuplicateAndClear()

			assert.ElementsMatch(t, tt.expectedBatch, duplicated)
			assert.Empty(t, tt.batcher.batch)
		})
	}
}

// This test is not a table test because I don't want to be misleading: This setup doesn't work for all cases and I can't figure out how to make it work for everything, so I'd rather make two cases.
func TestBatcher_Consume_HappyPath(t *testing.T) {
	expectedMessages := []monitor.Message{
		{StatusCode: 1},
		{StatusCode: 2},
		{StatusCode: 3},
		{StatusCode: 4},
		{StatusCode: 5},
		{StatusCode: 6},
	}

	ctx, cancel := context.WithCancel(context.Background())

	messageQueue := make(chan monitor.Message)

	var wg sync.WaitGroup

	fakeWriteOrLog := func() WriteOrLog {
		return func(_ context.Context, _ *sql.DB, batch []monitor.Message) {
			assert.ElementsMatch(t, batch, expectedMessages)
			wg.Done()
		}
	}

	batcher := &Batcher{
		mut:        &sync.Mutex{},
		batch:      []monitor.Message{},
		batchSize:  6,
		writeOrLog: fakeWriteOrLog(),
	}

	// this is a fake producer
	for _, message := range expectedMessages {
		wg.Add(1)

		go func() {
			messageQueue <- monitor.Message{
				StatusCode: message.StatusCode,
			}

			wg.Done()
		}()
	}

	wg.Add(1) // fakeWriteOrLog should happen only once, when the 6 elements are there.

	go batcher.Consume(ctx, messageQueue)

	wg.Wait() // wait for everything to finish
	cancel()
}

func TestBatcher_Consume_CancelBefore(t *testing.T) {
	expectedMessages := []monitor.Message{
		{StatusCode: 1},
		{StatusCode: 2},
		{StatusCode: 3},
		{StatusCode: 4},
		{StatusCode: 5},
		{StatusCode: 6},
	}

	ctx, cancel := context.WithCancel(context.Background())

	messageQueue := make(chan monitor.Message)

	var wg sync.WaitGroup

	var writeResult []monitor.Message

	fakeWriteOrLog := func() WriteOrLog {
		return func(_ context.Context, _ *sql.DB, batch []monitor.Message) {
			writeResult = batch // store the result of the write
		}
	}

	batcher := &Batcher{
		mut:        &sync.Mutex{},
		batch:      []monitor.Message{},
		batchSize:  6,
		writeOrLog: fakeWriteOrLog(),
	}

	// this is a fake producer
	for _, message := range expectedMessages {
		wg.Go(func() {
			messageQueue <- monitor.Message{
				StatusCode: message.StatusCode,
			}
		})
	}

	go batcher.Consume(ctx, messageQueue)

	wg.Wait() // wait for everything to finish
	cancel()

	assert.Empty(t, writeResult)
}
