package diagnostics

import (
	"context"
	"crypto/rsa"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/openshift/openshift-azure/pkg/api"
	"github.com/openshift/openshift-azure/pkg/util/azureclient"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/compute"
	"github.com/openshift/openshift-azure/pkg/util/azureclient/insights"
)

type scalesetDebugger struct {
	resourceGroup  string
	subscriptionID string
	testConfig     api.TestConfig
	log            *logrus.Entry
	ssc            compute.VirtualMachineScaleSetsClient
}

// NewScalesetDebugger create a new scaleset debugger
func NewScalesetDebugger(log *logrus.Entry, subscriptionID, resourceGroup string, testConfig api.TestConfig, ssc compute.VirtualMachineScaleSetsClient) *scalesetDebugger {
	return &scalesetDebugger{
		log:            log,
		resourceGroup:  resourceGroup,
		subscriptionID: subscriptionID,
		testConfig:     testConfig,
		ssc:            ssc,
	}
}

func (ssd *scalesetDebugger) GatherHostLogs(ctx context.Context, ssName string, vmCount int64, sshkey *rsa.PrivateKey) {
	s, err := NewSSHer(ctx, ssd.log, ssd.subscriptionID, ssd.resourceGroup, sshkey)
	if err != nil {
		ssd.log.Warnf("NewSSHer err %v", err)
		return
	}

	for i := int64(0); i < vmCount; i++ {
		hostname := fmt.Sprintf("%s-%06s", ssName[3:], strconv.FormatInt(i, 36))
		cli, err := s.Dial(ctx, hostname)
		if err != nil {
			ssd.log.Warnf("Dial failed: %v", err)
			continue
		}

		err = s.RunRemoteCommandAndSaveToFile(cli, "sudo journalctl", ssd.testConfig.ArtifactDir+"/"+hostname+"-early-journal")
		if err != nil {
			ssd.log.Warnf("RunRemoteCommandAndSaveToFile failed: %v", err)
			continue
		}

		err = s.RunRemoteCommandAndSaveToFile(cli, "sudo cat /var/lib/waagent/custom-script/download/1/stdout", ssd.testConfig.ArtifactDir+"/"+hostname+"-waagent-stdout")
		if err != nil {
			ssd.log.Warnf("RunRemoteCommandAndSaveToFile failed: %v", err)
			continue
		}

		err = s.RunRemoteCommandAndSaveToFile(cli, "sudo cat /var/lib/waagent/custom-script/download/1/stderr", ssd.testConfig.ArtifactDir+"/"+hostname+"-waagent-stderr")
		if err != nil {
			ssd.log.Warnf("RunRemoteCommandAndSaveToFile failed: %v", err)
			continue
		}
	}
}

func (ssd *scalesetDebugger) GatherActivityLogs(ctx context.Context) {
	authorizer, err := azureclient.GetAuthorizerFromContext(ctx, api.ContextKeyClientAuthorizer)
	if err != nil {
		ssd.log.Warnf("authorizer err %v", err)
		return
	}

	startTime := time.Now().Add(time.Duration(-1) * time.Hour)
	alc := insights.NewActivityLogsClient(ctx, ssd.log, ssd.subscriptionID, authorizer)
	logs, err := alc.List(ctx, fmt.Sprintf("eventTimestamp ge '%s' and resourceGroupName eq '%s'", startTime.Format(time.RFC3339), ssd.resourceGroup), "eventName,id,operationName,status")
	if err != nil {
		ssd.log.Warnf("alc.List err %v", err)
		return
	}

	logFile, err := os.OpenFile(path.Join(ssd.testConfig.ArtifactDir, "activity.log"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		ssd.log.Warnf("os.OpenFile err %v", err)
		return
	}
	defer logFile.Close()

	for logs.NotDone() {
		for _, log := range logs.Values() {
			lb, err := log.MarshalJSON()
			if err == nil {
				fmt.Fprintf(logFile, "%v\n", string(lb))
			}
		}
		err = logs.Next()
		if err != nil {
			break
		}
	}
}

func (ssd *scalesetDebugger) GatherStatuses(ctx context.Context, ssName string) {
	ss, err := ssd.ssc.Get(ctx, ssd.resourceGroup, ssName)
	if err != nil {
		ssd.log.Warnf("ssc.Get err %v", err)
		return
	}
	if ssd.subscriptionID == "" {
		ssd.subscriptionID = strings.Split(*ss.ID, "/")[2]
	}
	ssd.log.Debugf("%s status:%s, provisioning state:%s", *ss.Name, ss.Status, *ss.VirtualMachineScaleSetProperties.ProvisioningState)
	instView, err := ssd.ssc.GetInstanceView(ctx, ssd.resourceGroup, *ss.Name)
	if err != nil {
		ssd.log.Warnf("GetInstanceView err %v", err)
		return
	}
	logFile, err := os.OpenFile(path.Join(ssd.testConfig.ArtifactDir, ssName+"-statuses.log"), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		ssd.log.Warnf("os.OpenFile err %v", err)
		return
	}
	defer logFile.Close()

	if instView.Statuses != nil {
		for _, status := range *instView.Statuses {
			var ds, msg string
			if status.DisplayStatus != nil {
				ds = *status.DisplayStatus
			}
			if status.Message != nil {
				msg = *status.Message
			}
			fmt.Fprintf(logFile, "Statuses: displayStatus:%s, msg:%s\n", ds, msg)
		}
	}
	if instView.VirtualMachine != nil {
		for _, status := range *instView.VirtualMachine.StatusesSummary {
			fmt.Fprintf(logFile, "VirtualMachine: code:%s count:%d\n", *status.Code, *status.Count)
		}
	}
	if instView.Extensions != nil {
		for _, extView := range *instView.Extensions {
			for _, status := range *extView.StatusesSummary {
				fmt.Fprintf(logFile, "Extensions: %s, status: %s %d\n", *extView.Name, *status.Code, *status.Count)
			}
		}
	}
}
