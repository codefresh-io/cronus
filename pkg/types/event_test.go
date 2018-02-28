package types

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stretchr/testify/mock"
)

type CronguruMock struct {
	mock.Mock
}

func (m *CronguruMock) DescribeCronExpression(expression string) (string, error) {
	args := m.Called(expression)
	return args.String(0), args.Error(1)
}

func TestConstructEvent(t *testing.T) {
	type args struct {
		uri         string
		secret      string
		expression  string
		description string
	}
	tests := []struct {
		name         string
		args         args
		want         *Event
		wantErr      bool
		failDescribe bool
	}{
		{
			name: "construct valid event",
			args: args{
				uri:         "cron:codefresh:5 0 * 8 *:test-message:abcdef1234",
				secret:      "1234",
				expression:  "5 0 * 8 *",
				description: "At 00:05 in August",
			},
			want: &Event{
				Expression:  "5 0 * 8 *",
				Message:     "test-message",
				Account:     "abcdef1234",
				Secret:      "1234",
				Description: "At 00:05 in August",
				Status:      "active",
				Help:        commonHelp,
			},
		},
		{
			name: "fail to describe event",
			args: args{
				uri:         "cron:codefresh:5 0 * 8 *:test-message:abcdef1234",
				secret:      "1234",
				expression:  "5 0 * 8 *",
				description: "",
			},
			want: &Event{
				Expression:  "5 0 * 8 *",
				Account:     "abcdef1234",
				Message:     "test-message",
				Secret:      "1234",
				Description: "failed to get cron description",
				Status:      "active",
				Help:        commonHelp,
			},
			failDescribe: true,
		},
		{
			name: "invalid cron expression",
			args: args{
				uri: "cron:codefresh:invalid-expression:test-message:abcdef1234",
			},
			wantErr: true,
		},
		{
			name: "invalid cron uri (bad format)",
			args: args{
				uri: "bad:codefresh:5 0 * 8 *",
			},
			wantErr: true,
		},
		{
			name: "invalid cron uri (prefix 1)",
			args: args{
				uri: "bad:codefresh:5 0 * 8 *:test-message",
			},
			wantErr: true,
		},
		{
			name: "invalid cron uri (prefix 2)",
			args: args{
				uri: "cron:bad:5 0 * 8 *:test-message",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cronguru := new(CronguruMock)
			if !tt.wantErr {
				if tt.failDescribe {
					cronguru.On("DescribeCronExpression", tt.args.expression).Return(tt.args.description, errors.New("Test Error"))
				} else {
					cronguru.On("DescribeCronExpression", tt.args.expression).Return(tt.args.description, nil)
				}
			}
			got, err := ConstructEvent(tt.args.uri, tt.args.secret, cronguru)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConstructEvent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConstructEvent() = %v, want %v", got, tt.want)
			}
			// assert calls
			cronguru.AssertExpectations(t)
		})
	}
}
