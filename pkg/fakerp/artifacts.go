package fakerp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func GatherArtifacts(artifactDir, artifactKubeconfig string) error {
	config, err := clientcmd.BuildConfigFromFlags("", artifactKubeconfig)
	if err != nil {
		return err
	}
	kc, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	// gather node info
	if err := gatherNodes(kc, artifactDir); err != nil {
		return err
	}

	// gather pods from all namespaces
	// TODO: Ensure we don't leak any secrets. Fix either one of the following:
	// https://github.com/openshift/openshift-azure/issues/567
	// https://github.com/openshift/openshift-azure/issues/687
	// if err := gatherPods(kc, artifactDir); err != nil {
	//	return err
	// }

	// gather events from all namespaces
	if err := gatherEvents(kc, artifactDir); err != nil {
		return err
	}

	// gather control plane logs
	ns := "kube-system"
	if err := gatherLogs(kc, artifactDir, ns, "sync-master-000000"); err != nil {
		return err
	}
	// TODO: Get logs from the api server and etcd dynamically by using the master count
	if err := gatherLogs(kc, artifactDir, ns, "master-etcd-master-000000"); err != nil {
		return err
	}
	if err := gatherLogs(kc, artifactDir, ns, "master-etcd-master-000001"); err != nil {
		return err
	}
	if err := gatherLogs(kc, artifactDir, ns, "master-etcd-master-000002"); err != nil {
		return err
	}
	if err := gatherLogs(kc, artifactDir, ns, "master-api-master-000000"); err != nil {
		return err
	}
	if err := gatherLogs(kc, artifactDir, ns, "master-api-master-000001"); err != nil {
		return err
	}
	if err := gatherLogs(kc, artifactDir, ns, "master-api-master-000002"); err != nil {
		return err
	}
	// the controller manager uses leader election so only the leader can do writes.
	// Find out who is the leader and get its logs.
	cm, err := kc.CoreV1().ConfigMaps(ns).Get("kube-controller-manager", metav1.GetOptions{})
	if err != nil {
		return err
	}
	type leader struct {
		Holder string `json:"holderIdentity"`
	}
	var l leader
	if err := json.Unmarshal([]byte(cm.Annotations["control-plane.alpha.kubernetes.io/leader"]), &l); err != nil {
		return err
	}
	cmLeader := fmt.Sprintf("controllers-%s", strings.Split(l.Holder, "_")[0])
	return gatherLogs(kc, artifactDir, ns, cmLeader)
}

func gatherNodes(kc *kubernetes.Clientset, artifactDir string) error {
	nodeBuf := bytes.NewBuffer(nil)
	nodes, err := kc.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, node := range nodes.Items {
		b, err := yaml.Marshal(node)
		if err != nil {
			return err
		}
		if _, err := nodeBuf.Write(b); err != nil {
			return err
		}
		if _, err := nodeBuf.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return ioutil.WriteFile(filepath.Join(artifactDir, "nodes.yaml"), nodeBuf.Bytes(), 0777)
}

func gatherPods(kc *kubernetes.Clientset, artifactDir string) error {
	podBuf := bytes.NewBuffer(nil)
	pods, err := kc.CoreV1().Pods("").List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, pod := range pods.Items {
		b, err := yaml.Marshal(pod)
		if err != nil {
			return err
		}
		if _, err := podBuf.Write(b); err != nil {
			return err
		}
		if _, err := podBuf.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return ioutil.WriteFile(filepath.Join(artifactDir, "pods.yaml"), podBuf.Bytes(), 0777)
}

func gatherEvents(kc *kubernetes.Clientset, artifactDir string) error {
	eventBuf := bytes.NewBuffer(nil)
	events, err := kc.CoreV1().Events("").List(metav1.ListOptions{})
	if err != nil {
		return err
	}
	for _, event := range events.Items {
		b, err := yaml.Marshal(event)
		if err != nil {
			return err
		}
		if _, err := eventBuf.Write(b); err != nil {
			return err
		}
		if _, err := eventBuf.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return ioutil.WriteFile(filepath.Join(artifactDir, "events.yaml"), eventBuf.Bytes(), 0777)
}

func gatherLogs(kc *kubernetes.Clientset, artifactDir, ns, name string) error {
	log, err := kc.CoreV1().Pods(ns).GetLogs(name, &v1.PodLogOptions{}).DoRaw()
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(artifactDir, fmt.Sprintf("%s_%s.log", ns, name)), log, 0777)
}
