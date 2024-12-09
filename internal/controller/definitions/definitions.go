package definitions

const (
	CertsuiteRunStatusPhaseCertSuiteDeploying     = "CertSuiteDeploying"
	CertsuiteRunStatusPhaseCertSuiteDeployFailure = "CertSuiteDeployFailure"
	CertsuiteRunStatusCertSuiteRunning            = "CertSuiteRunning"
	CertsuiteRunStatusPhaseJobFinished            = "CertSuiteFinished"
	CertsuiteRunStatusPhaseJobError               = "CertSuiteError"
	CertsuiteRunStatusPhaseJobTimeout             = "CertSuiteTimeout"
)

const (
	CnfCertPodNamePrefix             = "certsuite-job-run"
	CnfCertSuiteSidecarContainerName = "certsuite-sidecar"
	CnfCertSuiteContainerName        = "certsuite"

	CnfCertSuiteBaseFolder      = "/certsuite"
	CnfCnfCertSuiteConfigFolder = CnfCertSuiteBaseFolder + "/config/suite"
	CnfPreflightConfigFolder    = CnfCertSuiteBaseFolder + "/config/preflight"
	CnfCertSuiteResultsFolder   = CnfCertSuiteBaseFolder + "/results"

	CnfCertSuiteConfigFilePath    = CnfCnfCertSuiteConfigFolder + "/tnf_config.yaml"
	PreflightDockerConfigFilePath = CnfPreflightConfigFolder + "/preflight_dockerconfig.json"

	SideCarResultsFolderEnvVar = "TNF_RESULTS_FOLDER"
	SideCarImageEnvVar         = "SIDECAR_APP_IMG"
	ControllerNamespaceEnvVar  = "CONTROLLER_NS"
)
