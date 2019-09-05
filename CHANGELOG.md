## v8.0

- Release v6 was removed  ([#1907](https://github.com/openshift/openshift-azure/pull/1907), [@troy0820](https://github.com/troy0820), 04/09/2019)
- Microsoft: when using logrus.SetReportCaller() you can now use the following to convert absolute file names in logs to relative ones.
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:    true,
		CallerPrettyfier: utillog.RelativeFilePathPrettier,
	})
 ([#1895](https://github.com/openshift/openshift-azure/pull/1895), [@asalkeld](https://github.com/asalkeld), 28/08/2019)
- Enabling boot diagnostics means serial console logs are available as
well as serial console access. ([#1875](https://github.com/openshift/openshift-azure/pull/1875), [@thekad](https://github.com/thekad), 26/08/2019)
- Allow the following to be set using the admin API:
- SSHSourceAddressPrefixes
- SecurityPatchPackages
- ComponentLogLevel ([#1876](https://github.com/openshift/openshift-azure/pull/1876), [@asalkeld](https://github.com/asalkeld), 23/08/2019)
- Microsoft: renamed api from 2019-08-31 to 2019-09-30-preview as requested. ([#1874](https://github.com/openshift/openshift-azure/pull/1874), [@asalkeld](https://github.com/asalkeld), 22/08/2019)
- Release v5 was removed  ([#1866](https://github.com/openshift/openshift-azure/pull/1866), [@troy0820](https://github.com/troy0820), 19/08/2019)
- Disable DisableOutboundSNAT for VMSS ([#1854](https://github.com/openshift/openshift-azure/pull/1854), [@mjudeikis](https://github.com/mjudeikis), 16/08/2019)
- Delay VM reboot after security updates to prevent possible race with Kubelet's startup. ([#1833](https://github.com/openshift/openshift-azure/pull/1833), [@charlesakalugwu](https://github.com/charlesakalugwu), 16/08/2019)
- Send logs to a customer's log analytics workspace ([#1812](https://github.com/openshift/openshift-azure/pull/1812), [@asalkeld](https://github.com/asalkeld), 15/08/2019)


