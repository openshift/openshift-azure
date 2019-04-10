package reporters

// GinkgoLogger suppresses logging unless "-v" is given to the gingko framework.
type AzureLogger struct{}

// GinkgoLogger implements Logger.
//var _ logrus.StdLogger = &AzureLogger{}
