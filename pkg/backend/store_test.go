package backend

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/codefresh-io/cronus/pkg/types"

	"github.com/stretchr/testify/assert"
)

func setupTestCase(t *testing.T) (func(t *testing.T), string) {
	t.Log("setup test case")
	dataDir, err := ioutil.TempDir("", "bolt")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("directory '%s' created", dataDir)
	eventsDB := filepath.Join(dataDir, "events.db")
	return func(t *testing.T) {
		t.Log("teardown test case")
		os.RemoveAll(dataDir)
	}, eventsDB
}

// func setupSubTest(t *testing.T) func(t *testing.T) {
// 	t.Log("setup sub test")
// 	return func(t *testing.T) {
// 		t.Log("teardown sub test")
// 	}
// }

func TestBoltEventStore_StoreEvent(t *testing.T) {
	type args struct {
		event types.Event
	}
	tests := []struct {
		name     string
		args     args
		expected int
		wantErr  bool
	}{
		{
			name: "store single event",
			args: args{types.Event{
				Expression:  "5 4 * * *",
				Message:     "test-message",
				Secret:      "1234",
				Description: "At 04:05",
				Status:      "active",
				Help:        "help",
			}},
			expected: 1,
		},
	}
	// setup and tear down
	teardownTestCase, eventsDB := setupTestCase(t)
	defer teardownTestCase(t)

	b, err := NewBoltEventStore(eventsDB)
	if err != nil {
		t.Fatal(err)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := b.StoreEvent(tt.args.event); (err != nil) != tt.wantErr {
				t.Errorf("BoltEventStore.StoreEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
			// get number of records
			records, err := b.GetDBStats()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if records != tt.expected {
				t.Errorf("Unexpected number of records %v != %v", records, tt.expected)
			}
		})
	}
}

func TestBoltEventStore_DeleteEvent(t *testing.T) {
	type args struct {
		uri string
	}
	tests := []struct {
		name     string
		args     args
		events   []types.Event
		expected int
		wantErr  bool
	}{
		{
			name: "delete existing event",
			args: args{uri: "cron:codefresh:5 4 * * *:test-message-1:abcdef1234"},
			events: []types.Event{
				{
					Expression:  "5 4 * * *",
					Message:     "test-message-1",
					Account:     "abcdef1234",
					Secret:      "1234",
					Description: "At 04:05",
					Status:      "active",
					Help:        "help",
				},
				{
					Expression:  "5 0 * 8 *",
					Message:     "test-message-2",
					Account:     "abcdef1234",
					Secret:      "1234",
					Description: "At 00:05 in August",
					Status:      "active",
					Help:        "help",
				},
			},
			expected: 1,
		},
		{
			name: "fail to delete non-existing event",
			args: args{uri: "cron:codefresh:1 1 * * *:test-message:abcd1234"},
			events: []types.Event{
				{
					Expression:  "5 4 * * *",
					Message:     "test-message-1",
					Secret:      "1234",
					Account:     "abcd1234",
					Description: "At 04:05",
					Status:      "active",
					Help:        "help",
				},
				{
					Expression:  "5 0 * 8 *",
					Message:     "test-message-2",
					Secret:      "1234",
					Account:     "abcd5678",
					Description: "At 00:05 in August",
					Status:      "active",
					Help:        "help",
				},
			},
			expected: 2,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup and tear down the test case
			teardownTestCase, eventsDB := setupTestCase(t)
			defer teardownTestCase(t)
			b, err := NewBoltEventStore(eventsDB)
			if err != nil {
				t.Fatal(err)
			}
			// create events
			for _, e := range tt.events {
				b.StoreEvent(e)
			}
			// invoke delete
			if err := b.DeleteEvent(tt.args.uri); (err != nil) != tt.wantErr {
				t.Errorf("BoltEventStore.DeleteEvent() error = %v, wantErr %v", err, tt.wantErr)
			}
			// get number of records
			records, err := b.GetDBStats()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if records != tt.expected {
				t.Errorf("Unexpected number of records %v != %v", records, tt.expected)
			}
		})
	}
}

func TestBoltEventStore_GetEvent(t *testing.T) {
	type args struct {
		uri string
	}
	tests := []struct {
		name    string
		events  []types.Event
		args    args
		want    *types.Event
		wantErr bool
	}{
		{
			name: "get existing event",
			args: args{uri: "cron:codefresh:5 0 * 8 *:test-message-2:abcd1234"},
			events: []types.Event{
				{
					Expression:  "5 4 * * *",
					Message:     "test-message-1",
					Secret:      "1234",
					Account:     "abcd1234",
					Description: "At 04:05",
					Status:      "active",
					Help:        "help",
				},
				{
					Expression:  "5 0 * 8 *",
					Message:     "test-message-2",
					Secret:      "1234",
					Account:     "abcd1234",
					Description: "At 00:05 in August",
					Status:      "active",
					Help:        "help",
				},
			},
			want: &types.Event{
				Expression:  "5 0 * 8 *",
				Message:     "test-message-2",
				Secret:      "1234",
				Account:     "abcd1234",
				Description: "At 00:05 in August",
				Status:      "active",
				Help:        "help",
			},
		},
		{
			name: "fail to get non-existing event",
			args: args{uri: "cron:codefresh:1 1 * * *:test-message:abcd1234"},
			events: []types.Event{
				{
					Expression:  "5 4 * * *",
					Message:     "test-message-1",
					Secret:      "1234",
					Account:     "abcd1234",
					Description: "At 04:05",
					Status:      "active",
					Help:        "help",
				},
				{
					Expression:  "5 0 * 8 *",
					Message:     "test-message-2",
					Secret:      "1234",
					Account:     "abcd5678",
					Description: "At 00:05 in August",
					Status:      "active",
					Help:        "help",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup and tear down the test case
			teardownTestCase, eventsDB := setupTestCase(t)
			defer teardownTestCase(t)
			b, err := NewBoltEventStore(eventsDB)
			if err != nil {
				t.Fatal(err)
			}
			// create events
			for _, e := range tt.events {
				b.StoreEvent(e)
			}
			// invoke
			got, err := b.GetEvent(tt.args.uri)
			if (err != nil) != tt.wantErr {
				t.Errorf("BoltEventStore.GetEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BoltEventStore.GetEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBoltEventStore_GetAllEvents(t *testing.T) {
	tests := []struct {
		name    string
		events  []types.Event
		want    []types.Event
		wantErr bool
	}{
		{
			name: "get all existing event",
			events: []types.Event{
				{
					Expression:  "5 4 * * *",
					Message:     "test-message-1",
					Secret:      "1234",
					Description: "At 04:05",
					Status:      "active",
					Help:        "help",
				},
				{
					Expression:  "5 0 * 8 *",
					Message:     "test-message-2",
					Secret:      "1234",
					Description: "At 00:05 in August",
					Status:      "active",
					Help:        "help",
				},
			},
			want: []types.Event{
				{
					Expression:  "5 4 * * *",
					Message:     "test-message-1",
					Secret:      "1234",
					Description: "At 04:05",
					Status:      "active",
					Help:        "help",
				},
				{
					Expression:  "5 0 * 8 *",
					Message:     "test-message-2",
					Secret:      "1234",
					Description: "At 00:05 in August",
					Status:      "active",
					Help:        "help",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// setup and tear down the test case
			teardownTestCase, eventsDB := setupTestCase(t)
			defer teardownTestCase(t)
			b, err := NewBoltEventStore(eventsDB)
			if err != nil {
				t.Fatal(err)
			}
			// create events
			for _, e := range tt.events {
				b.StoreEvent(e)
			}
			// invoke
			got, err := b.GetAllEvents()
			if (err != nil) != tt.wantErr {
				t.Errorf("BoltEventStore.GetAllEvents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.ElementsMatch(t, got, tt.want, "unexpected result")
		})
	}
}
