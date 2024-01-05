package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type DockerError struct {
	Code    string      `json:"code"`
	Message string      `json:"message,omitempty"`
	Detail  interface{} `json:"detail,omitempty"`
}

type DockerAuthError struct {
	DockerError
	AuthRequest WwwAuthRequest
}

type DockerResponse struct {
	Name                string `json:"name,omitempty"`
	DockerContentDigest string
	Tags                []string      `json:"tags,omitempty"`
	Repositories        []string      `json:"repositories,omitempty"`
	Errors              []DockerError `json:"errors,omitempty"`
}

func (e DockerError) Error() string {
	return e.Message
}

func (e DockerAuthError) Error() string {
	return e.Message
}

func callDockerRepo(method, url, authHeader string, headers map[string]string) (*DockerResponse, error) {
	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		fmt.Println("Error creating new request:", err)
		return nil, fmt.Errorf("error fetching tags: %s", err)
	}

	req.Header.Add("Authorization", authHeader)
	for key, val := range headers {
		req.Header.Add(key, val)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error fetching tags:", err)
		return nil, fmt.Errorf("error fetching tags: %s", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, fmt.Errorf("error reading response body: %s", err)
	}

	if resp.Header.Get("Www-Authenticate") != "" {
		wwwAuth := resp.Header.Get("Www-Authenticate")
		if wwwAuth != "" {
			wwwAuth, err := parseWwwAuthenticateHeader(wwwAuth)
			if err == nil {
				return nil, &DockerAuthError{
					AuthRequest: wwwAuth,
				}
			}
		}
	}

	var response DockerResponse
	if len(body) != 0 {
		err = json.Unmarshal(body, &response)
		if err != nil {
			fmt.Println("Error parsing JSON response:", err)
			return nil, fmt.Errorf("error parsing JSON response: %s", err)
		}
	}
	response.DockerContentDigest = resp.Header.Get("Docker-Content-Digest")

	if len(response.Errors) > 0 {
		wwwAuth := resp.Header.Get("Www-Authenticate")
		if wwwAuth != "" {
			wwwAuth, err := parseWwwAuthenticateHeader(wwwAuth)
			if err == nil {
				return nil, &DockerAuthError{
					DockerError: response.Errors[0],
					AuthRequest: wwwAuth,
				}
			}
		}

		return nil, response.Errors[0]
	}

	return &response, nil
}
