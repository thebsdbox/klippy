package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	log "github.com/Sirupsen/logrus"

	"github.com/docker/docker/client"
	"github.com/spf13/cobra"
	"github.com/thebsdbox/klippy/pkg/registry"
)

// The Used to set the logging level
var loglevel int

// Docker Client, pointer so that we can use it as nil to determine if docker is running
var dockerClient *client.Client

var imageName string

var klippyCmd = &cobra.Command{
	Use:   "klippy",
	Short: "klippy",
}

func init() {
	// Global flag across all subcommands
	klippyCmd.PersistentFlags().IntVar(&loglevel, "logLevel", 4, "Set the logging level [0=panic, 3=warning, 5=debug]")
	imageLookup.PersistentFlags().StringVar(&imageName, "name", "", "")
	imageLookup.AddCommand(tagLookup)
	imageLookup.AddCommand(cmdLookup)
	imageLookup.AddCommand(overviewLookup)

	klippyCmd.AddCommand(imageLookup)

	// log.Info("Starting environment initialisation and inspection")
	// log.Info("Looking for Docker endpoint")
	// var err error
	// dockerClient, err = client.NewClientWithOpts(client.WithVersion("1.38"))
	// if err != nil {
	// 	log.Warnf("%s", err.Error())
	// } else {
	// 	v, err := dockerClient.ServerVersion(context.Background())
	// 	if err != nil {
	// 		log.Warnf("%s", err.Error())
	// 	} else {
	// 		log.Infof("Found Docker Version [%s]", v.APIVersion)
	// 	}
	// }
}

// Execute - Start the CLI evaluation
func Execute() {
	log.SetLevel(log.Level(loglevel))
	if err := klippyCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var imageLookup = &cobra.Command{
	Use:   "image",
	Short: "Lookup information about an image",
	Run: func(cmd *cobra.Command, args []string) {
		log.SetLevel(log.Level(loglevel))
		cmd.Help()
	},
}

var tagLookup = &cobra.Command{
	Use:   "tags",
	Short: "List all tags of a specific image",
	Run: func(cmd *cobra.Command, args []string) {
		log.SetLevel(log.Level(loglevel))
		if imageName == "" {
			cmd.Help()
			log.Fatalf("No image specified")
		}
		tags, err := registry.RetrieveTags(imageName)
		if err != nil {
			log.Fatalf("%s", err.Error())
		}

		for i := range tags {
			fmt.Printf("\t%s\n", tags[i])
		}
	},
}

var cmdLookup = &cobra.Command{
	Use:   "commands",
	Short: "List all commands used to build a specific image",
	Run: func(cmd *cobra.Command, args []string) {
		log.SetLevel(log.Level(loglevel))
		if imageName == "" {
			cmd.Help()
			log.Fatalf("No image specified")
		}
		cmds, err := registry.RetrieveCommands(imageName)
		if err != nil {
			log.Fatalf("%s", err.Error())
		}
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)

		fmt.Fprintln(w, "Layer\tCommand")

		for i := range cmds {
			fmt.Fprintf(w, "\033[37m%d\t%s\n", i, cmds[i])
		}
		w.Flush()
	},
}

var overviewLookup = &cobra.Command{
	Use:   "overview",
	Short: "Print an overview of the details of a specific image",
	Run: func(cmd *cobra.Command, args []string) {
		log.SetLevel(log.Level(loglevel))
		if imageName == "" {
			cmd.Help()
			log.Fatalf("No image specified")
		}
		manifest, err := registry.RetrieveOverview(imageName)
		if err != nil {
			log.Fatalf("%s", err.Error())
		}
		fmt.Printf("Name:\t%s\n", manifest.Name)
		fmt.Printf("Arch:\t%s\n", manifest.Architecture)
		fmt.Printf("Tag:\t%s\n", manifest.Tag)
		fmt.Println("Layers:")
		for i := range manifest.FsLayers {
			fmt.Printf("\tLayer [%d]:\t%s\n", i, manifest.FsLayers[i].BlobSum)
		}
		// fmt.Println("Layer Build")
		// for i := range manifest.History {
		// 	var v1Layer registry.V1ContainerLayer
		// 	err = json.Unmarshal([]byte(manifest.History[i].V1Compatibility), &v1Layer)
		// 	if err != nil {
		// 		log.Fatalf("%s", err.Error())
		// 	}
		// 	fmt.Printf("\tLayer Architecture:\t%s\n", v1Layer.DockerVersion)
		// }
	},
}
