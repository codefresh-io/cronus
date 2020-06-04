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
			want: "2020-06-05 04:05:00 +0300 EEST",
		},
		{
			name: "complex expression",
			args: args{expression: "23 0-20/2 * * *"},
			want: "2020-06-04 18:23:00 +0300 EEST",
		},
		{
			name: "@weekly expression",
			args: args{expression: "@weekly"},
			want: "2020-06-07 00:00:00 +0300 EEST",
		},
		{
			name: "@annually expression",
			args: args{expression: "@annually"},
			want: "2021-01-01 00:00:00 +0200 EET",
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
			expr := NewCronExpression()
			got, err := expr.DescribeCronExpression(tt.args.expression)
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
