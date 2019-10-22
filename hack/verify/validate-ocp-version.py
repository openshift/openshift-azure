#!/usr/bin/python3

import re
import sys
import yaml

PLUGIN_CONFIG = 'pluginconfig/pluginconfig-311.yaml'


def main():
    config = yaml.load(open(PLUGIN_CONFIG, 'r'))
    tag_re = re.compile(r':v3\.11\.(\d+)')

    for version in config['versions'].values():
        vm_version = version['imageVersion']
        vm_ocp_version = vm_version.split('.')[1]
        for container_version in version['images'].values():
            tag = tag_re.search(container_version)
            if 'registry.access.redhat.com/openshift3' in container_version and tag and tag[1] != vm_ocp_version:
                print('VM version {vm} and container tag {ct} do not match'.format(vm=vm_version, ct=container_version))
                sys.exit(1)


if __name__ == '__main__':
    main()
