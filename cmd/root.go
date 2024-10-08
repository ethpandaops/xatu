package cmd

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	log       = logrus.New()
	logLevel  string
	logFormat string
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
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "", "Log level (debug, info, warn, error, fatal, panic)")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "", "Log format (text, json)")
}

func initCommon() {
	if logFormat == "json" {
		log.SetFormatter(&logrus.JSONFormatter{})
	} else {
		log.SetFormatter(&logrus.TextFormatter{})
	}

	if logLevel != "" {
		logLevel, err := logrus.ParseLevel(logLevel)
		if err != nil {
			log.Fatalf("invalid logging level: %v", err)
		}

		log.SetLevel(logLevel)
	}
}

func getLoggerWithOverride(overrideLevel, overrideFormat string) *logrus.Logger {
	if overrideLevel != "" {
		logLevel, err := logrus.ParseLevel(strings.ToLower(overrideLevel))
		if err != nil {
			log.Fatalf("invalid logging level: %v", err)
		}

		log.SetLevel(logLevel)
	}

	if overrideFormat != "" {
		if overrideFormat == "text" {
			log.SetFormatter(&logrus.TextFormatter{})
		} else {
			log.SetFormatter(&logrus.JSONFormatter{})
		}
	}

	log.WithField("level", log.GetLevel()).Info("Logger initialized")

	return log
}
