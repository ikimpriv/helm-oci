package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strings"
)

type WwwAuthRequest struct {
	Realm   string
	Service string
	Scope   string
}

type DockerConfig struct {
	CredsStore  string               `json:"credsStore"`
	CredHelpers map[string]string    `json:"credHelpers"`
	Auths       map[string]AuthEntry `json:"auths"`
}

type HelmConfig struct {
	Auths map[string]AuthEntry `json:"auths"`
}

type AuthEntry struct {
	Auth string `json:"auth"`
}

type DockerCredential struct {
	Username  string `json:"Username"`
	Secret    string `json:"Secret"`
	ServerURL string `json:"ServerURL"`
}

func getCredentials(serverURL string) (username, password string, err error) {
	helmConfig, err := parseHelmConfig()
	if err == nil {
		if entry, ok := helmConfig.Auths[serverURL]; ok {
			return decodeAuth(entry.Auth)
		}
	}

	dockerConfig, err := parseDockerConfig()
	if err == nil {
		if entry, ok := dockerConfig.Auths[serverURL]; ok {
			username, password, err = decodeAuth(entry.Auth)
			if err == nil {
				return username, password, err
			}
		}

		helper := dockerConfig.CredsStore
		if specificHelper, ok := dockerConfig.CredHelpers[serverURL]; ok {
			helper = specificHelper
		}

		if helper != "" {
			return getCredentialsFromHelper(helper, serverURL)
		}
	}

	return "", "", fmt.Errorf("credentials not found for %s", serverURL)
}

func parseHelmConfig() (*HelmConfig, error) {
	helmConfigPath := os.Getenv("HELM_CONFIG_HOME")
	if helmConfigPath != "" {
		return parseHelmConfigFile(path.Join(helmConfigPath, "registry, config.json"))
	}

	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	return parseHelmConfigFile(fmt.Sprintf("%s/helm/registry/config.json", configDir))
}

func parseHelmConfigFile(helmConfigFile string) (*HelmConfig, error) {
	data, err := os.ReadFile(helmConfigFile)
	if err != nil {
		return nil, err
	}

	var cfg HelmConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func parseDockerConfig() (cfg *DockerConfig, err error) {
	dockerConfigPath := os.Getenv("DOCKER_CONFIG")
	if dockerConfigPath != "" {
		return parseDockerConfigFile(path.Join(dockerConfigPath, "config.json"))
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	cfg, err = parseDockerConfigFile(fmt.Sprintf("%s/.docker/config.json", homeDir))
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func parseDockerConfigFile(configFile string) (*DockerConfig, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var cfg DockerConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func decodeAuth(encodedAuth string) (username, password string, err error) {
	decoded, err := base64.StdEncoding.DecodeString(encodedAuth)
	if err != nil {
		return "", "", err
	}

	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", "", errors.New("invalid auth data")
	}

	return parts[0], parts[1], nil
}

func getCredentialsFromHelper(helper, serverURL string) (username, password string, err error) {
	cmd := exec.Command(fmt.Sprintf("docker-credential-%s", helper), "get")
	cmd.Stdin = strings.NewReader(serverURL)
	output, err := cmd.Output()
	if err != nil {
		if strings.HasPrefix(string(output), "credentials not found") {
			return "", "", fmt.Errorf("credentials not found for %s", serverURL)
		}
		return "", "", err
	}

	var creds DockerCredential
	if err := json.Unmarshal(output, &creds); err != nil {
		return "", "", err
	}

	return creds.Username, creds.Secret, nil
}

func retrieveToken(wwwAuthHeader WwwAuthRequest, username, password string) (string, error) {
	client := &http.Client{}

	tokenReq, err := http.NewRequest(http.MethodGet, wwwAuthHeader.Realm, nil)
	if err != nil {
		fmt.Println("Error creating new request:", err)
		return "", fmt.Errorf("error fetching token: %s", err)
	}

	tokenReq.SetBasicAuth(username, password)

	query := tokenReq.URL.Query()
	query.Add("service", wwwAuthHeader.Service)
	query.Add("scope", wwwAuthHeader.Scope)
	tokenReq.URL.RawQuery = query.Encode()

	resp, err := client.Do(tokenReq)
	if err != nil {
		fmt.Println("Error fetching token:", err)
		return "", fmt.Errorf("error fetching token: %s", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return "", fmt.Errorf("error reading response body: %s", err)
	}

	var tokenResponse TokenResponseStruct

	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		fmt.Println("Error parsing JSON response:", err)
		return "", fmt.Errorf("error parsing JSON response: %s", err)
	}

	return tokenResponse.Token, nil
}

func parseWwwAuthenticateHeader(wwwAuth string) (WwwAuthRequest, error) {
	authHeader := WwwAuthRequest{}
	authFields := wwwAuth[7:] //Stripping 'Bearer ' part of header
	params := strings.Split(authFields, ",")

	for _, param := range params {
		field := strings.SplitN(param, "=", 2)
		if len(field) < 2 {
			return WwwAuthRequest{}, errors.New("malformed WWW-Authenticate header")
		}
		key := strings.TrimSpace(field[0])

		// Removing quotes from value
		valueMatch, _ := regexp.MatchString(`"(.+?)"`, field[1])
		value := field[1]
		if valueMatch {
			value = strings.Trim(field[1], "\"")
		}

		switch key {
		case "realm":
			authHeader.Realm = value
		case "service":
			authHeader.Service = value
		case "scope":
			authHeader.Scope = value
		}
	}

	return authHeader, nil
}
