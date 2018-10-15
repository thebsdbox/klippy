package registry

// This is a bit of a learning exercise, useful URLs below
// : https://docs.docker.com/registry/spec/api/#errors-2
// : https://github.com/moby/moby/blob/master/contrib/download-frozen-image-v2.sh
//

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
)

type tagsStruct struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

//v2Manifest contains the manifest that defines a container image
type v2Manifest struct {
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

type v1ContainerLayer struct {
	ID              string    `json:"id"`
	Parent          string    `json:"parent"`
	Created         time.Time `json:"created"`
	ContainerConfig struct {
		Cmd []string `json:"Cmd"`
	} `json:"container_config"`
	Throwaway bool `json:"throwaway"`
}

func httpGet(url, authToken string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// If a token has been added
	if len(authToken) != 0 {
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", authToken))
	}

	req.Header.Add("Content-Type", "application/json")

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

//identifyRegistry will break apart the image url and look for the base
func identifyRegistryImageTag(imageurl string) (registry, image, tag string, err error) {

	u, err := url.Parse(imageurl)
	if err != nil {
		return "", "", "", err
	}

	// Check if a protocol is defined, if not set it to https
	if u.Scheme == "" {
		u, err = url.Parse("https://" + imageurl)
		if err != nil {
			return "", "", "", err
		}
		log.Debugf("Reparsing modified URL [%s]", u.String())
	}

	_, err = net.LookupHost(u.Hostname())
	if err != nil {
		log.Debugf("Unable to resolve [%s] dropping back to docker hub", u.Hostname())
		u, err = url.Parse("https://registry-1.docker.io/" + imageurl)
		if err != nil {
			return "", "", "", err
		}
		log.Debugf("Reparsing modified URL [%s]", u.String())
	}

	registry = strings.TrimSuffix(u.String(), u.Path)
	// Split the NameSpace:tag by the colon delimiter
	parts := strings.Split(u.Path, ":")

	// Parse the output from splitting the url path
	if len(parts) > 1 {
		// Expected output
		image = parts[0]
		tag = parts[1]
	}
	// If only a namespace/image is specified drop to the latest tag (docker behaviour)
	if len(parts) == 1 {
		log.Debugf("Setting tag to \"latest\"")
		tag = "latest"
	}
	if len(parts) > 2 {
		log.Warnf("Expecting only 2 parts to Namespace/project : tag")
	}
	if len(parts) == 0 {
		return "", "", "", fmt.Errorf("Unable to parse namespace/image:tag")
	}
	image = parts[0]

	log.Debugf("Identified registry [%s]", registry)

	// Remove the first character from the image as it will be a slash [1:]
	return registry, image[1:], tag, nil
}

// identifyRegistryAuthBearer - hits the v2 url and find the bearer server, and returns a token
func identifyRegistryAuthBearer(registry, image string) (string, error) {

	//Registry /v2 endpoint
	v2Endpoint := registry + "/v2/"
	response, err := httpGet(v2Endpoint, "")
	if err != nil {
		return "", err
	}
	apiVersion := response.Header.Get("Docker-Distribution-API-Version")
	if apiVersion == "" {
		log.Warnln("Unknown registry version")
	}
	log.Debugf("Registry version [%s]", apiVersion)

	//Locate the bearer server
	wwwAuthResponse := response.Header.Get("WWW-Authenticate")
	if wwwAuthResponse == "" {
		return "", fmt.Errorf("Unknown Auth Header")
	}
	log.Debugf("WWW-Authenticate header [%s]", wwwAuthResponse)
	bearerURL, bearerService := getBearerSettings(wwwAuthResponse)
	if bearerURL == "" {
		return "", fmt.Errorf("No Registry bearer server could be identified for registry [%s]", registry)
	}
	if bearerService == "" {
		return "", fmt.Errorf("No Registry bearer service could be identified for registry [%s]", registry)
	}
	authURL := fmt.Sprintf("%s?service=%s&scope=repository:%s:pull", bearerURL, bearerService, image)
	log.Debugf("Built URL [%s]", authURL)

	// Close the previous ioreader
	response.Body.Close()

	// Perform the HTTP Get against the authorisation server
	response, err = httpGet(authURL, "")
	if err != nil {
		return "", err
	}
	// Close this response at the end of the function
	defer response.Body.Close()

	// Read the contents of the response into a []byte
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	// Struct for the auth server response
	var authResponse struct {
		Token string `json:"token"`
		// TODO - other json objects are returned
	}

	// Parse contents into a struct
	err = json.Unmarshal(body, &authResponse)
	if err != nil {
		return "", err
	}
	if authResponse.Token == "" {
		return "", fmt.Errorf("No Token could be identified in the response from the authorisation server")
	}
	log.Debugf("Token of [%d] bytes found", len(authResponse.Token))
	return authResponse.Token, nil
}

// This will parse the header and find the v2 registry details needed
func getBearerSettings(headerString string) (registryRealm, registryService string) {

	parts := strings.SplitN(headerString, " ", 2)
	// split KV by comma
	parts = strings.Split(parts[1], ",")
	// iterate over the KV values
	for _, part := range parts {
		// split through the Key = Value
		vals := strings.SplitN(part, "=", 2)
		// Assign K/V
		key := vals[0]
		value := strings.Trim(vals[1], "\",")
		log.Debugf("Header Value:[%s] Key:[%s]", key, value)
		if key == "realm" {
			registryRealm = value
		}
		if key == "service" {
			registryService = value
		}
	}
	// Returns "" if no realm is found
	return registryRealm, registryService
}

func retrieveImageTags(registry, image, token string) ([]string, error) {
	//Build the Registry v2 URL
	v2RegistryURL := fmt.Sprintf("%s/v2/%s/tags/list", registry, image)
	log.Debugf("Built v2 Registry URL [%s]", v2RegistryURL)
	response, err := httpGet(v2RegistryURL, token)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != 200 {
		log.Debugf("HTTP Error [%s]", response.Status)
		return nil, fmt.Errorf("Unable to retrieve tags for image [%s]", image)
	}
	// Close this response at the end of the function
	defer response.Body.Close()

	// Read the contents of the response into a []byte
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var tagList tagsStruct
	err = json.Unmarshal(body, &tagList)

	return tagList.Tags, nil
}

func retrieveImageCommands(registry, image, tag, token string) ([]string, error) {
	//Build the Registry v2 URL
	v2RegistryURL := fmt.Sprintf("%s/v2/%s/manifests/%s", registry, image, tag)
	log.Debugf("Built v2 Registry URL [%s]", v2RegistryURL)
	response, err := httpGet(v2RegistryURL, token)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != 200 {
		log.Debugf("HTTP Error [%s]", response.Status)
		return nil, fmt.Errorf("Unable to retrieve tags for image [%s]", image)
	}
	// Close this response at the end of the function
	defer response.Body.Close()

	// Read the contents of the response into a []byte
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	//fmt.Printf("%s", body)

	// Unmarshall the Manifest json
	var manifest v2Manifest
	err = json.Unmarshal(body, &manifest)
	if err != nil {
		return nil, err
	}

	// Iterate over the v1 Layers
	var commands []string
	for i := range manifest.History {
		var layer v1ContainerLayer

		err = json.Unmarshal([]byte(manifest.History[i].V1Compatibility), &layer)
		if err != nil {
			return nil, err
		}
		// Combine the string, from the string array
		var buildString string
		buildString = "\033[32m"
		for x := range layer.ContainerConfig.Cmd {
			buildString = fmt.Sprintf(("%s%s"), buildString, layer.ContainerConfig.Cmd[x])
		}

		// Find if a NOP (No Operation scenario exists)
		nop := strings.Split(buildString, "#(nop) ")
		if len(nop) > 1 {
			buildString = fmt.Sprintf("\033[31m%s\033[0m", strings.TrimSpace(nop[len(nop)-1]))
		}
		// Sanitise from the usually bonkers amounts of tabs
		tabSanitise := strings.Replace(buildString, "\t", "", -1)
		// Tidy the newlines
		ampersandSanitsie := strings.Replace(tabSanitise, "&&", "\\\n       \033[37m&&\033[32m", -1)
		commands = append(commands, ampersandSanitsie)
	}
	return commands, nil
}
