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
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_wait"
)

func TestForHTTPStatusOk(t *testing.T) {
	urltocheck := "http://localhost:12345/nowhere"
	logger := logrus.New()
	log := logrus.NewEntry(logger)

	type cliResp struct {
		err  error
		resp *http.Response
	}

	tests := []struct {
		name      string
		responses []cliResp
		err       error
		wantErr   bool
	}{
		{
			name:      "working",
			responses: []cliResp{{resp: &http.Response{StatusCode: 200}}},
		},
		{
			name: "ENETUNREACH then ok",
			responses: []cliResp{
				{err: &url.Error{
					URL: urltocheck,
					Err: &net.OpError{
						Err: os.NewSyscallError("socket", syscall.ENETUNREACH),
					},
				}},
				{resp: &http.Response{StatusCode: 200}},
			},
		},
		{
			name: "ECONNREFUSED then ok",
			responses: []cliResp{
				{err: &url.Error{
					URL: urltocheck,
					Err: &net.OpError{
						Err: os.NewSyscallError("socket", syscall.ECONNREFUSED),
					},
				}},
				{resp: &http.Response{StatusCode: 200}},
			},
		},
		{
			name: "io.EOF then ok",
			responses: []cliResp{
				{err: io.EOF},
				{resp: &http.Response{StatusCode: 200}},
			},
		},
		{
			name: "url io.EOF then ok",
			responses: []cliResp{
				{err: &url.Error{Err: io.EOF}},
				{resp: &http.Response{StatusCode: 200}},
			},
		},
		{
			name: "url io.ErrUnexpectedEOF then ok",
			responses: []cliResp{
				{err: &url.Error{Err: io.ErrUnexpectedEOF}},
				{resp: &http.Response{StatusCode: 200}},
			},
		},
		{
			name: "500 then ok",
			responses: []cliResp{
				{resp: &http.Response{StatusCode: 500}},
				{resp: &http.Response{StatusCode: 200}},
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
	gmc := gomock.NewController(t)
	defer gmc.Finish()

	for _, tt := range tests {
		mockCli := mock_wait.NewMockSimpleHTTPClient(gmc)
		req, _ := http.NewRequest("GET", urltocheck, nil)
		for _, resp := range tt.responses {
			mockCli.EXPECT().Do(req).Return(resp.resp, resp.err)
		}

		_, err := ForHTTPStatusOk(context.Background(), log, mockCli, urltocheck, time.Nanosecond)
		if tt.wantErr != (err != nil) || tt.wantErr && tt.err.Error() != err.Error() {
			t.Errorf("forHTTPStatusOk(%s) error = %v, Err %v", tt.name, err, tt.err)
		}
	}
}
