package cronexp

import (
	"testing"
)

func TestAPIEndpoint_DescribeCronExpression(t *testing.T) {
	type args struct {
		expression string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "basic expression",
			args: args{expression: "5 4 * * *"},
			want: "At 04:05 AM",
		},
		{
			name: "complex expression",
			args: args{expression: "23 0-20/2 * * *"},
			want: "At 23 minutes past the hour, every 2 hours, between 12:00 AM and 08:59 PM",
		},
		{
			name: "@weekly expression",
			args: args{expression: "@weekly"},
			want: "At 12:00 AM, only on Sunday",
		},
		{
			name: "@annually expression",
			args: args{expression: "@annually"},
			want: "At 12:00 AM, on day 1 of the month, only in January",
		},
		{
			name:    "bad expression",
			args:    args{expression: "hello"},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := NewCronDescriptorEndpoint()
			got, err := api.DescribeCronExpression(tt.args.expression)
			if (err != nil) != tt.wantErr {
				t.Errorf("APIEndpoint.DescribeCronExpression() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("APIEndpoint.DescribeCronExpression() = %v, want %v", got, tt.want)
			}
		})
	}
}
