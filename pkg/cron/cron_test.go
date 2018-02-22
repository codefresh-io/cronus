package cron

import (
	"errors"
	"sync"
	"testing"

	"github.com/codefresh-io/cronus/pkg/hermes"
	"github.com/codefresh-io/cronus/pkg/types"
	"github.com/stretchr/testify/mock"
	cron "gopkg.in/robfig/cron.v2"
)

// StoreMock mock
type StoreMock struct {
	mock.Mock
}

func (m *StoreMock) StoreEvent(event types.Event) error {
	args := m.Called(event)
	return args.Error(0)
}

func (m *StoreMock) DeleteEvent(uri string) error {
	args := m.Called(uri)
	return args.Error(0)
}

func (m *StoreMock) GetEvent(uri string) (*types.Event, error) {
	args := m.Called(uri)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.Event), args.Error(1)
}

func (m *StoreMock) GetAllEvents() ([]types.Event, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.Event), args.Error(1)
}

func (m *StoreMock) GetDBStats() (int, error) {
	args := m.Called()
	return args.Int(0), args.Error(1)
}

// CronJobEngineMock
type CronJobEngineMock struct {
	mock.Mock
}

func (m *CronJobEngineMock) Start() {
	m.Called()
}

func (m *CronJobEngineMock) AddJob(spec string, cmd cron.Job) (cron.EntryID, error) {
	args := m.Called(spec, cmd)
	return cron.EntryID(args.Int(0)), args.Error(1)
}

func (m *CronJobEngineMock) Remove(id cron.EntryID) {
	m.Called(id)
}

// HermesMock
type HermesMock struct {
	mock.Mock
}

func (m *HermesMock) TriggerEvent(eventURI string, event *hermes.NormalizedEvent) error {
	args := m.Called(eventURI, event)
	return args.Error(0)
}

func TestNewCronRunnerFull(t *testing.T) {
	type expected struct {
		events []types.Event
	}
	tests := []struct {
		name     string
		expected expected
	}{
		{
			name: "happy path",
			expected: expected{
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
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storeMock := &StoreMock{}
			cronJobMock := &CronJobEngineMock{}
			hermesMock := &HermesMock{}
			// mock store
			storeMock.On("GetAllEvents").Return(tt.expected.events, nil)
			// mock cron engine calls
			for i, e := range tt.expected.events {
				cronJobMock.On("AddJob", e.Expression, mock.Anything).Return(i, nil)
			}
			// mock start
			cronJobMock.On("Start")
			// invoke
			NewCronRunnerFull(storeMock, hermesMock, cronJobMock)
			// assert
			storeMock.AssertExpectations(t)
			cronJobMock.AssertExpectations(t)
			hermesMock.AssertExpectations(t)
		})
	}
}

func TestRunner_triggerEvent(t *testing.T) {
	type args struct {
		e types.Event
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "trigger event",
			args: args{
				e: types.Event{
					Expression:  "5 4 * * *",
					Message:     "test-message-1",
					Secret:      "1234",
					Description: "At 04:05",
					Status:      "active",
					Help:        "help",
				},
			},
		},
		{
			name: "trigger event with account",
			args: args{
				e: types.Event{
					Expression:  "5 4 * * *",
					Message:     "test-message-1",
					Account:     "cb1e73c5215b",
					Secret:      "1234",
					Description: "At 04:05",
					Status:      "active",
					Help:        "help",
				},
			},
		},
		{
			name: "fail to trigger event",
			args: args{
				e: types.Event{
					Expression:  "5 4 * * *",
					Message:     "test-message-1",
					Secret:      "1234",
					Description: "At 04:05",
					Status:      "active",
					Help:        "help",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hermesMock := &HermesMock{}
			r := &Runner{
				hermesSvc: hermesMock,
			}
			// mock hermes call
			call := hermesMock.On("TriggerEvent", types.GetURI(tt.args.e), mock.AnythingOfType("*hermes.NormalizedEvent"))
			if tt.wantErr {
				call.Return(errors.New("Test Error"))
			} else {
				call.Return(nil)
			}
			// invoke
			r.TriggerEvent(tt.args.e)
			// assert
			hermesMock.AssertExpectations(t)
		})
	}
}

func TestRunner_AddCronJob(t *testing.T) {
	type args struct {
		e types.Event
	}
	tests := []struct {
		name             string
		args             args
		wantAddJobErr    bool
		wantjobExistsErr bool
		wantStoreError   bool
	}{
		{
			name: "add cron event job",
			args: args{
				e: types.Event{
					Expression:  "5 4 * * *",
					Message:     "test-message-1",
					Secret:      "1234",
					Description: "At 04:05",
					Status:      "active",
					Help:        "help",
				},
			},
		},
		{
			name: "add cron event job with account",
			args: args{
				e: types.Event{
					Expression:  "5 4 * * *",
					Message:     "test-message-1",
					Account:     "cb1e73c5215b",
					Secret:      "1234",
					Description: "At 04:05",
					Status:      "active",
					Help:        "help",
				},
			},
		},
		{
			name: "job already exists",
			args: args{
				e: types.Event{
					Expression:  "5 4 * * *",
					Message:     "test-message-1",
					Secret:      "1234",
					Description: "At 04:05",
					Status:      "active",
					Help:        "help",
				},
			},
			wantjobExistsErr: true,
		},
		{
			name: "fail AddJob",
			args: args{
				e: types.Event{
					Expression:  "5 4 * * *",
					Message:     "test-message-1",
					Secret:      "1234",
					Description: "At 04:05",
					Status:      "active",
					Help:        "help",
				},
			},
			wantAddJobErr: true,
		},
		{
			name: "fail StoreEvent",
			args: args{
				e: types.Event{
					Expression:  "5 4 * * *",
					Message:     "test-message-1",
					Secret:      "1234",
					Description: "At 04:05",
					Status:      "active",
					Help:        "help",
				},
			},
			wantStoreError: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var call *mock.Call
			storeMock := &StoreMock{}
			cronMock := &CronJobEngineMock{}
			r := &Runner{
				store: storeMock,
				cron:  cronMock,
				jobs:  new(sync.Map),
			}
			// job already exists?
			if tt.wantjobExistsErr {
				r.jobs.Store(types.GetURI(tt.args.e), 1)
				goto Invoke
			}
			// mock cron job
			call = cronMock.On("AddJob", tt.args.e.Expression, mock.Anything)
			if tt.wantAddJobErr {
				call.Return(0, errors.New("failed to create a new cron job"))
				goto Invoke
			} else {
				call.Return(1, nil)
			}
			// mock store
			call = storeMock.On("StoreEvent", tt.args.e)
			if tt.wantStoreError {
				call.Return(errors.New("Test Error"))
				cronMock.On("Remove", cron.EntryID(1))
			} else {
				call.Return(nil)
			}
			// invoke
		Invoke:
			if err := r.AddCronJob(tt.args.e); (err != nil) != (tt.wantAddJobErr || tt.wantjobExistsErr || tt.wantStoreError) {
				t.Errorf("Runner.AddCronJob() error = %v, wantErr %v", err, tt.wantAddJobErr || tt.wantjobExistsErr || tt.wantStoreError)
			}
			// assert calls
			storeMock.AssertExpectations(t)
			cronMock.AssertExpectations(t)
		})
	}
}

func TestRunner_RemoveCronJob(t *testing.T) {
	type args struct {
		uri string
	}
	tests := []struct {
		name               string
		args               args
		wantDeleteErr      bool
		wantjobNotExistErr bool
	}{
		{
			name: "delete cron event",
			args: args{uri: "cron:codefresh:5 4 * * *:test-message"},
		},
		{
			name:               "delete non-existing event",
			args:               args{uri: "cron:codefresh:5 4 * * *:test-message"},
			wantjobNotExistErr: true,
		},
		{
			name:          "deleteEvent error",
			args:          args{uri: "cron:codefresh:5 4 * * *:test-message"},
			wantDeleteErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var call *mock.Call
			storeMock := &StoreMock{}
			cronMock := &CronJobEngineMock{}
			r := &Runner{
				store: storeMock,
				cron:  cronMock,
				jobs:  new(sync.Map),
			}
			// add job to map in no error expected
			if tt.wantjobNotExistErr {
				goto Invoke
			} else {
				r.jobs.Store(tt.args.uri, cron.EntryID(1))
			}
			// mock cron calls
			cronMock.On("Remove", cron.EntryID(1))
			// mock store calls
			call = storeMock.On("DeleteEvent", tt.args.uri)
			if tt.wantDeleteErr {
				call.Return(errors.New("Test Error"))
			} else {
				call.Return(nil)
			}
			// invoke
		Invoke:
			if err := r.RemoveCronJob(tt.args.uri); (err != nil) != (tt.wantDeleteErr || tt.wantjobNotExistErr) {
				t.Errorf("Runner.RemoveCronJob() error = %v, wantErr %v", err, (tt.wantDeleteErr || tt.wantjobNotExistErr))
			}
			// assert
			cronMock.AssertExpectations(t)
			storeMock.AssertExpectations(t)
		})
	}
}
