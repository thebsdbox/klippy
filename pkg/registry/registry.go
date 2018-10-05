package registry

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	log "github.com/Sirupsen/logrus"
)

// ImageExists is used to determine if a docker image actually exists
func ImageExists(c *client.Client, imageName string) (bool, error) {

	// Results list pinned to 25
	results, err := c.ImageSearch(context.Background(), imageName, types.ImageSearchOptions{Limit: 25})
	if err != nil {
		return false, err
	}
	if len(results) > 1 {
		log.Infof("There are [%d] results for the search term [%s]", len(results), imageName)
		for i := range results {
			fmt.Printf("%s\n", results[i].Name)
		}
	}

	return false, nil
}
