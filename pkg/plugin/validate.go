// Package plugin holds the implementation of a plugin.
package plugin

import "fmt"

func (p *plugin) validateConfig() (errs []error) {
	if p.config.SyncImage == "" {
		errs = append(errs, fmt.Errorf("syncImage cannot be empty"))
	}
	if len(p.config.GenevaConfig.ImagePullSecret) == 0 {
		errs = append(errs, fmt.Errorf("imagePullSecret cannot be empty"))
	}
	if p.config.GenevaConfig.LoggingCert == nil {
		errs = append(errs, fmt.Errorf("loggingCert cannot be nil"))
	}
	if p.config.GenevaConfig.LoggingKey == nil {
		errs = append(errs, fmt.Errorf("loggingKey cannot be nil"))
	}
	if p.config.GenevaConfig.LoggingSector == "" {
		errs = append(errs, fmt.Errorf("loggingSector cannot be empty"))
	}
	if p.config.GenevaConfig.LoggingImage == "" {
		errs = append(errs, fmt.Errorf("loggingImage cannot be empty"))
	}
	if p.config.GenevaConfig.LoggingAccount == "" {
		errs = append(errs, fmt.Errorf("loggingAccount cannot be empty"))
	}
	if p.config.GenevaConfig.LoggingNamespace == "" {
		errs = append(errs, fmt.Errorf("loggingNamespace cannot be empty"))
	}
	if p.config.GenevaConfig.LoggingControlPlaneAccount == "" {
		errs = append(errs, fmt.Errorf("loggingControlPlaneAccount cannot be empty"))
	}
	if p.config.GenevaConfig.TDAgentImage == "" {
		errs = append(errs, fmt.Errorf("tdAgentImage cannot be empty"))
	}
	if p.config.GenevaConfig.MetricsCert == nil {
		errs = append(errs, fmt.Errorf("metricsCert cannot be nil"))
	}
	if p.config.GenevaConfig.MetricsKey == nil {
		errs = append(errs, fmt.Errorf("metricsKey cannot be nil"))
	}
	if p.config.GenevaConfig.MetricsBridge == "" {
		errs = append(errs, fmt.Errorf("metricsBridge cannot be empty"))
	}
	if p.config.GenevaConfig.StatsdImage == "" {
		errs = append(errs, fmt.Errorf("statsdImage cannot be empty"))
	}
	if p.config.GenevaConfig.MetricsAccount == "" {
		errs = append(errs, fmt.Errorf("metricsAccount cannot be empty"))
	}
	if p.config.GenevaConfig.MetricsEndpoint == "" {
		errs = append(errs, fmt.Errorf("metricsEndpoint cannot be empty"))
	}
	return errs
}
