package registry

// This is a bit of a learning exercise, useful URLs below
// : https://docs.docker.com/registry/spec/api/#detail
// : https://github.com/moby/moby/blob/master/contrib/download-frozen-image-v2.sh
//

import (
	"fmt"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
)

type tagsStruct struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

//V2Manifest contains the manifest that defines a container image
type V2Manifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	Name          string `json:"name"`
	Tag           string `json:"tag"`
	Architecture  string `json:"architecture"`
	FsLayers      []struct {
		BlobSum string `json:"blobSum"`
	} `json:"fsLayers"`
	History []struct {
		V1Compatibility string `json:"v1Compatibility"`
	} `json:"history"`
	Signatures []struct {
		Header struct {
			Jwk struct {
				Crv string `json:"crv"`
				Kid string `json:"kid"`
				Kty string `json:"kty"`
				X   string `json:"x"`
				Y   string `json:"y"`
			} `json:"jwk"`
			Alg string `json:"alg"`
		} `json:"header"`
		Signature string `json:"signature"`
		Protected string `json:"protected"`
	} `json:"signatures"`
}

//V1ContainerLayer - Contains the information about a Container layer
type V1ContainerLayer struct {
	Architecture string `json:"architecture"`
	Config       struct {
		Hostname     string      `json:"Hostname"`
		Domainname   string      `json:"Domainname"`
		User         string      `json:"User"`
		AttachStdin  bool        `json:"AttachStdin"`
		AttachStdout bool        `json:"AttachStdout"`
		AttachStderr bool        `json:"AttachStderr"`
		Tty          bool        `json:"Tty"`
		OpenStdin    bool        `json:"OpenStdin"`
		StdinOnce    bool        `json:"StdinOnce"`
		Env          []string    `json:"Env"`
		Cmd          []string    `json:"Cmd"`
		Image        string      `json:"Image"`
		Volumes      interface{} `json:"Volumes"`
		WorkingDir   string      `json:"WorkingDir"`
		Entrypoint   interface{} `json:"Entrypoint"`
		OnBuild      interface{} `json:"OnBuild"`
		Labels       struct {
		} `json:"Labels"`
	} `json:"config"`
	Container       string `json:"container"`
	ContainerConfig struct {
		Hostname     string      `json:"Hostname"`
		Domainname   string      `json:"Domainname"`
		User         string      `json:"User"`
		AttachStdin  bool        `json:"AttachStdin"`
		AttachStdout bool        `json:"AttachStdout"`
		AttachStderr bool        `json:"AttachStderr"`
		Tty          bool        `json:"Tty"`
		OpenStdin    bool        `json:"OpenStdin"`
		StdinOnce    bool        `json:"StdinOnce"`
		Env          []string    `json:"Env"`
		Cmd          []string    `json:"Cmd"`
		Image        string      `json:"Image"`
		Volumes      interface{} `json:"Volumes"`
		WorkingDir   string      `json:"WorkingDir"`
		Entrypoint   interface{} `json:"Entrypoint"`
		OnBuild      interface{} `json:"OnBuild"`
		Labels       struct {
		} `json:"Labels"`
	} `json:"container_config"`
	Created       time.Time `json:"created"`
	DockerVersion string    `json:"docker_version"`
	ID            string    `json:"id"`
	Os            string    `json:"os"`
	Parent        string    `json:"parent"`
	Throwaway     bool      `json:"throwaway"`
}

func httpGet(url, authToken string, v2 bool) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// If a token has been added
	if len(authToken) != 0 {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", authToken))
	}

	// Expecting JSON
	req.Header.Add("Content-Type", "application/json")

	if v2 == true {
		// Accept the newer version of the manifests
		req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
		req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.list.v2+json")
	}

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// RetrieveTags - This will find an image on a registry and return all its tags
func RetrieveTags(imageName string) ([]string, error) {
	// Split Image Name and locate if a registry is part of it exists
	registry, image, tag, err := identifyRegistryImageTag(imageName)
	log.Debugf("Registry [%s], Image[%s], Tag [%s]", registry, image, tag)
	if err != nil {
		return nil, err
	}

	token, err := identifyRegistryAuthBearer(registry, image)
	if err != nil {
		return nil, err
	}

	return retrieveImageTags(registry, image, token)
}

// RetrieveCommands - This will find an image on a registry and return all of it's commands
func RetrieveCommands(imageName string) ([]string, error) {
	// Split Image Name and locate if a registry is part of it exists
	registry, image, tag, err := identifyRegistryImageTag(imageName)
	log.Debugf("Registry [%s], Image[%s], Tag [%s]", registry, image, tag)
	if err != nil {
		return nil, err
	}

	token, err := identifyRegistryAuthBearer(registry, image)
	if err != nil {
		return nil, err
	}

	return retrieveImageCommands(registry, image, tag, token)
}

// RetrieveOverview - This will find an image on a registry and return all of it's commands
func RetrieveOverview(imageName string) (*V2Manifest, error) {
	// Split Image Name and locate if a registry is part of it exists
	registry, image, tag, err := identifyRegistryImageTag(imageName)
	log.Debugf("Registry [%s], Image[%s], Tag [%s]", registry, image, tag)
	if err != nil {
		return nil, err
	}

	token, err := identifyRegistryAuthBearer(registry, image)
	if err != nil {
		return nil, err
	}

	return retrieveImageOverview(registry, image, tag, token)
}

// ImageExists is used to determine if a docker image actually exists
func ImageExists(imageName string) (bool, error) {
	log.Infof("Beginning lookup of image [%s]", imageName)

	// Split Image Name and locate if a registry is part of it exists
	registry, image, tag, err := identifyRegistryImageTag(imageName)
	log.Debugf("Registry [%s], Image[%s], Tag [%s]", registry, image, tag)
	if err != nil {
		return false, err
	}

	_, err = identifyRegistryAuthBearer(registry, image)
	if err != nil {
		return false, err
	}

	return true, nil
}
