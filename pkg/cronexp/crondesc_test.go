package cronexp

import (
	"testing"
	"time"
)

func TestAPIEndpoint_DescribeCronExpression(t *testing.T) {
	type args struct {
		expression string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "basic expression",
			args: args{expression: "5 4 * * * ?"},
		},
		{
			name: "complex expression",
			args: args{expression: "23 0-20/2 * * * ?"},
		},
		{
			name: "@weekly expression",
			args: args{expression: "@weekly"},
		},
		{
			name: "@annually expression",
			args: args{expression: "@annually"},
		},
		{
			name:    "bad expression",
			args:    args{expression: "hello"},
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
			if got == "" {
				return
			}

			_, err = time.Parse(time.RFC3339, got)
			if err != nil {
				t.Errorf("APIEndpoint.DescribeCronExpression() = %v. It is not a timestamp", got)
			}
		})
	}
}
