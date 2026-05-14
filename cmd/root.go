package cmd

import (
	"os"
	"strings"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	log           = newRootLogger()
	logLevelFlag  string
	logFormatFlag string

	defaultLogLevel  = "info"
	defaultLogFormat = "json"

	metricsAddrFlag = "metrics-addr"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "xatu",
	Short: "",
	Long:  ``,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	initCommon()

	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&logLevelFlag, "log-level", "", "Log level (debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().StringVar(&logFormatFlag, "log-format", "", "Log format (text, json)")
}

func initCommon() {
}

// newRootLogger constructs the package-level logger with the OTel
// trace-context hook attached so log entries carrying a span emit
// trace_id and span_id fields.
func newRootLogger() *logrus.Logger {
	l := logrus.New()

	observability.InstallTraceContextHook(l)

	return l
}

func getLogger(configLevel, configFormat string) *logrus.Logger {
	// Prefer the cli args over whatever has been provided in the config file
	finalLevel := defaultLogLevel
	if logLevelFlag != "" {
		finalLevel = logLevelFlag
	} else if configLevel != "" {
		finalLevel = configLevel
	}

	logLevel, err := logrus.ParseLevel(strings.ToLower(finalLevel))
	if err != nil {
		log.Fatalf("invalid logging level: %v", err)
	}

	log.SetLevel(logLevel)

	finalFormat := defaultLogFormat
	if logFormatFlag != "" {
		finalFormat = logFormatFlag
	} else if configFormat != "" {
		finalFormat = configFormat
	}

	if finalFormat == "json" {
		log.SetFormatter(&logrus.JSONFormatter{})
	} else {
		log.SetFormatter(&logrus.TextFormatter{})
	}

	log.WithFields(logrus.Fields{
		"level": log.GetLevel(),
	}).Info("Logger initialized")

	return log
}
