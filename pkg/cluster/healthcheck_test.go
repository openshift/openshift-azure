package cluster

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/mocks/mock_wait"
	"github.com/sirupsen/logrus"
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			ctx := context.Background()
			u := &simpleUpgrader{
				log: logrus.NewEntry(logrus.StandardLogger()),
			}
			urltocheck := "https://" + tt.cs.Properties.FQDN + "/console/"
			mockCli := mock_wait.NewMockSimpleHTTPClient(mockCtrl)
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
