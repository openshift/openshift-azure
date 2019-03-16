package cluster

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_wait"
)

func TestHealthCheck(t *testing.T) {
	type cliResp struct {
		err  error
		resp *http.Response
	}
	tests := []struct {
		name      string
		cs        *api.OpenShiftManagedCluster
		want      *api.PluginError
		responses []cliResp
	}{
		{
			name: "working",
			cs: &api.OpenShiftManagedCluster{
				Properties: api.Properties{
					FQDN: "notarealaddress",
				},
			},
			responses: []cliResp{{resp: &http.Response{StatusCode: 200}}},
		},
		{
			name: "errors, then ok",
			cs: &api.OpenShiftManagedCluster{
				Properties: api.Properties{
					FQDN: "notarealaddress",
				},
			},
			responses: []cliResp{
				{resp: &http.Response{StatusCode: http.StatusBadGateway}},
				{resp: &http.Response{StatusCode: 200}},
			},
		},
		{
			name: "bad status",
			cs: &api.OpenShiftManagedCluster{
				Properties: api.Properties{
					FQDN: "notarealaddress",
				},
			},
			responses: []cliResp{
				{resp: &http.Response{StatusCode: http.StatusBadRequest}},
			},
			want: &api.PluginError{Err: fmt.Errorf("unexpected error code %d from console", 400), Step: api.PluginStepWaitForConsoleHealth},
		},
		{
			name: "bad Do error",
			cs: &api.OpenShiftManagedCluster{
				Properties: api.Properties{
					FQDN: "notarealaddress",
				},
			},
			responses: []cliResp{
				{err: fmt.Errorf("bad thing")},
			},
			want: &api.PluginError{Err: fmt.Errorf("bad thing"), Step: api.PluginStepWaitForConsoleHealth},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gmc := gomock.NewController(t)
			defer gmc.Finish()
			ctx := context.Background()
			u := &simpleUpgrader{
				log: logrus.NewEntry(logrus.StandardLogger()),
			}
			urltocheck := "https://" + tt.cs.Properties.FQDN + "/console/"
			mockCli := mock_wait.NewMockSimpleHTTPClient(gmc)
			req, _ := http.NewRequest("HEAD", urltocheck, nil)
			req = req.WithContext(ctx)
			for _, resp := range tt.responses {
				mockCli.EXPECT().Do(req).Return(resp.resp, resp.err)
			}

			if got := u.doHealthCheck(ctx, mockCli, urltocheck, time.Nanosecond); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("simpleUpgrader.HealthCheck() = %v, want %v", got, tt.want)
			}
		})
	}
}
