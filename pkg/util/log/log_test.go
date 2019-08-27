package log

import (
	"runtime"
	"strings"
	"testing"
)

func TestRelativeFilePathPrettier(t *testing.T) {
	_, thisfile, _, _ := runtime.Caller(0)
	tests := []struct {
		f            *runtime.Frame
		wantFileName string
		wantFuncName string
	}{
		{
			f:            &runtime.Frame{File: thisfile, Line: 5, Function: "github.com/openshift/openshift-azure/pkg/util/log.TestRelativeFilePathPrettier"},
			wantFileName: "pkg/util/log/log_test.go:5",
			wantFuncName: "pkg/util/log.TestRelativeFilePathPrettier()",
		},
		{
			f: &runtime.Frame{
				File:     strings.Replace(thisfile, "pkg/util/log/log_test.go", "pkg/fakerp/customer_handlers.go", -1),
				Line:     89,
				Function: "github.com/openshift/openshift-azure/pkg/fakerp.(*Server).handlePut",
			},
			wantFileName: "pkg/fakerp/customer_handlers.go:89",
			wantFuncName: "pkg/fakerp.(*Server).handlePut()",
		},
	}
	for _, tt := range tests {
		t.Run(tt.wantFileName, func(t *testing.T) {
			gotFunc, gotFile := RelativeFilePathPrettier(tt.f)
			if gotFile != tt.wantFileName {
				t.Errorf("RelativeFilePathPrettier() got = %v, want %v", gotFile, tt.wantFileName)
			}
			if gotFunc != tt.wantFuncName {
				t.Errorf("RelativeFilePathPrettier() got = %v, want %v", gotFunc, tt.wantFuncName)
			}
		})
	}
}
