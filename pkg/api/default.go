package api

// DefaultVMSizeKubeArguments defines default values of kube-arguments based on the VM size
var DefaultVMSizeKubeArguments = map[VMSize]map[string]string{
	StandardD2sV3: {
		"kube-reserved":   "cpu=200m,memory=512Mi",
		"system-reserved": "cpu=200m,memory=512Mi",
	},
	StandardD4sV3: {
		"kube-reserved":   "cpu=500m,memory=512Mi",
		"system-reserved": "cpu=500m,memory=512Mi",
	},
}
