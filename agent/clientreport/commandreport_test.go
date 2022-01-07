package clientreport

import (
	"testing"

	"bou.ke/monkey"
)

func TestReportCommandOutput(t *testing.T) {
	type args struct {
		reportType string
		command    string
		arguments  []string
	}
	theArgs := args{
		reportType: "reportType",
		command: "echo",
		arguments: []string{"hello"},
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "normal",
			args: theArgs,
			want: "",
			wantErr: false,
		},
	}
	guard := monkey.Patch(SendReport, func(ClientReport) (string, error) { return "", nil} )
	defer guard.Unpatch()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReportCommandOutput(tt.args.reportType, tt.args.command, tt.args.arguments)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReportCommandOutput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ReportCommandOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}
