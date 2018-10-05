package cmd

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"github.com/thebsdbox/klippy/pkg/registry"
)

// The Used to set the logging level
var loglevel int

// Docker Client, pointer so that we can use it as nil to determine if docker is running
var dockerClient *client.Client

var klippyCmd = &cobra.Command{
	Use:   "klippy",
	Short: "klippy",
}

func init() {
	// Global flag across all subcommands
	klippyCmd.PersistentFlags().IntVar(&loglevel, "logLevel", 4, "Set the logging level [0=panic, 3=warning, 5=debug]")
	log.Info("Starting environment initialisation and inspection")
	log.Info("Looking for Docker endpoint")
	var err error
	dockerClient, err = client.NewClientWithOpts(client.WithVersion("1.38"))
	if err != nil {
		log.Warnf("%s", err.Error())
	} else {
		log.Infof("Found Docker Version [%s]", dockerClient.ClientVersion())
	}
}

// Execute - Start the CLI evaluation
func Execute() {
	log.SetLevel(log.Level(loglevel))
	_, err := registry.ImageExists(dockerClient, "thebsdbox/ovcli")
	if err != nil {
		log.Warnf("%s", err.Error())
	}
	if err := klippyCmd.Execute(); err != nil {
		klippyCmd.Help()
		fmt.Println(err)
		os.Exit(1)
	}
}
