package clientreport

import (
	"testing"

	gomonkey "github.com/agiledragon/gomonkey/v2"
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
	guard := gomonkey.ApplyFunc(SendReport, func(ClientReport) (string, error) { return "", nil} )
	defer guard.Reset()
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
