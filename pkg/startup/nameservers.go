package startup

import (
	"fmt"
	"io/ioutil"
	"path"
	"strings"
)

func getServerFromDNSConf(content string) ([]string, error) {
	servers := []string{}
	for _, line := range strings.Split(content, "\n") {
		if !strings.HasPrefix(line, "server=") {
			continue
		}
		servers = append(servers, strings.Split(line, "=")[1])
	}
	if len(servers) == 0 {
		return nil, fmt.Errorf("no servers found in origin-upstream-dns.conf")
	}
	return servers, nil
}

// GetNameserversFromDNSConfig read the nameservers from:
// {rootDir}/etc/dnsmasq.d/origin-upstream-dns.conf
func GetNameserversFromDNSConfig(rootDir string) ([]string, error) {
	dnsConfFile := path.Join(rootDir, "/etc/dnsmasq.d/origin-upstream-dns.conf")
	b, err := ioutil.ReadFile(dnsConfFile)
	if err != nil {
		return nil, err
	}
	return getServerFromDNSConf(string(b))
}
