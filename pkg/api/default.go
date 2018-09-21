package api

// DefaultVMSizeKubeArguments defines default values of kube-arguments based on the VM size
var DefaultVMSizeKubeArguments = map[VMSize]map[AgentPoolProfileRole]map[ReservedResource]string{
	StandardD2sV3: {
		AgentPoolProfileRoleMaster: {
			SystemReserved: "cpu=500m,memory=1Gi",
		},
		AgentPoolProfileRoleCompute: {
			KubeReserved:   "cpu=200m,memory=512Mi",
			SystemReserved: "cpu=200m,memory=512Mi",
		},
		AgentPoolProfileRoleInfra: {
			KubeReserved:   "cpu=200m,memory=512Mi",
			SystemReserved: "cpu=200m,memory=512Mi",
		},
	},
	StandardD4sV3: {
		AgentPoolProfileRoleMaster: {
			SystemReserved: "cpu=1000m,memory=1Gi",
		},
		AgentPoolProfileRoleCompute: {
			KubeReserved:   "cpu=500m,memory=512Mi",
			SystemReserved: "cpu=500m,memory=512Mi",
		},
		AgentPoolProfileRoleInfra: {
			KubeReserved:   "cpu=500m,memory=512Mi",
			SystemReserved: "cpu=500m,memory=512Mi",
		},
	},
}
