package wait

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"syscall"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_wait"
)

func TestForHTTPStatusOk(t *testing.T) {
	urltocheck := "http://localhost:12345/nowhere"

	var unreachableErr error = &url.Error{
		URL: urltocheck,
		Err: &net.OpError{
			Err: os.NewSyscallError("socket", syscall.ENETUNREACH),
		},
	}
	type cliResp struct {
		err  error
		resp http.Response
	}
	tests := []struct {
		name      string
		responses []cliResp
		err       error
		wantErr   bool
	}{
		{
			name:      "working",
			responses: []cliResp{{resp: http.Response{StatusCode: 200}}},
		},
		{
			name: "unreachableErr then ok",
			responses: []cliResp{
				{err: unreachableErr},
				{resp: http.Response{StatusCode: 200}},
			},
		},
		{
			name: "io.EOF then ok",
			responses: []cliResp{
				{err: io.EOF},
				{resp: http.Response{StatusCode: 200}},
			},
		},
		{
			name: "url io.EOF then ok",
			responses: []cliResp{
				{err: &url.Error{Err: io.EOF}},
				{resp: http.Response{StatusCode: 200}},
			},
		},
		{
			name: "url io.ErrUnexpectedEOF then ok",
			responses: []cliResp{
				{err: &url.Error{Err: io.ErrUnexpectedEOF}},
				{resp: http.Response{StatusCode: 200}},
			},
		},
		{
			name: "500 then ok",
			responses: []cliResp{
				{resp: http.Response{StatusCode: 500}},
				{resp: http.Response{StatusCode: 200}},
			},
		},
		{
			name: "unknown error",
			responses: []cliResp{
				{err: errors.New("oops")},
			},
			err:     errors.New("oops"),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		mockCtrl := gomock.NewController(t)
		defer mockCtrl.Finish()

		mockCli := mock_wait.NewMockSimpleHTTPClient(mockCtrl)
		iteration := 0
		returner := func(req *http.Request) (*http.Response, error) {
			resp := tt.responses[iteration]
			iteration++
			return &resp.resp, resp.err
		}

		req, _ := http.NewRequest("GET", urltocheck, nil)
		mockCli.EXPECT().Do(req).DoAndReturn(returner).Times(len(tt.responses))
		err := forHTTPStatusOkWithTimeout(context.Background(), mockCli, urltocheck)
		if tt.wantErr != (err != nil) || tt.wantErr && tt.err.Error() != err.Error() {
			t.Errorf("forHTTPStatusOkWithTimeout(%s) error = %v, Err %v", tt.name, err, tt.err)
		}
	}
}
