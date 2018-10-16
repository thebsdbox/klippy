package registry

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"strings"

	log "github.com/Sirupsen/logrus"
)

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

	// Split the NameSpace@sha:tag by the @ delimiter
	parts := strings.Split(u.Path, "@")
	// Parse the output from splitting the url path
	if len(parts) == 2 {
		// Expected output
		return registry, parts[0][1:], parts[1], nil
	}

	// Split the NameSpace:tag by the colon delimiter
	parts = strings.Split(u.Path, ":")

	// Parse the output from splitting the url path
	if len(parts) == 2 {
		// Expected output
		return registry, parts[0][1:], parts[1], nil
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
	response, err := httpGet(v2Endpoint, "", false)
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
	response, err = httpGet(authURL, "", false)
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
	response, err := httpGet(v2RegistryURL, token, false)
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

func retrieveImageOverview(registry, image, tag, token string) (*V2Manifest, error) {
	//Build the Registry v2 URL
	v2RegistryURL := fmt.Sprintf("%s/v2/%s/manifests/%s", registry, image, tag)
	log.Debugf("Built v2 Registry URL [%s]", v2RegistryURL)
	response, err := httpGet(v2RegistryURL, token, false)
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

	// Unmarshall the Manifest json
	var manifest V2Manifest
	err = json.Unmarshal(body, &manifest)
	if err != nil {
		return nil, err
	}

	return &manifest, nil
}

func retrieveImageCommands(registry, image, tag, token string) ([]string, error) {

	manifest, err := retrieveImageOverview(registry, image, tag, token)
	if err != nil {
		return nil, err
	}

	// Iterate over the v1 Layers
	var commands []string
	for i := range manifest.History {
		var layer V1ContainerLayer

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
