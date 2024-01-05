package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/mkideal/cli"
	"log"
	"os"
	"strings"
)

type rmT struct {
	cli.Helper
}

func (argv *rmT) Validate(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		return fmt.Errorf("usage: helm-repo rm <repository/path> <tag>")
	}
	repoParts := strings.SplitN(ctx.Args()[0], "/", 2)
	if len(repoParts) < 2 {
		return fmt.Errorf("usage: helm-repo rm <repository/path> <tag>")
	}
	return nil
}

var rm = &cli.Command{
	Name: "rm",
	Desc: "Delete a tag",
	Argv: func() interface{} { return new(rmT) },
	Fn: func(ctx *cli.Context) error {
		spec := ctx.Args()[0]
		tag := ctx.Args()[1]

		spec = strings.TrimPrefix(spec, "oci://")

		repoParts := strings.SplitN(spec, "/", 2)
		if len(repoParts) < 2 {
			return fmt.Errorf("the artifact spec is not defined")
		}
		repo := repoParts[0]
		path := repoParts[1]

		username, password, err := getCredentials(repo)
		if err != nil {
			log.Printf("Error: %v\n", err)
			os.Exit(-1)
		}

		digest, err := getManifestDigest(fmt.Sprintf("https://%s/v2/%s/manifests/%s", repo, path, tag), username, password)
		if err != nil {
			fmt.Println("can't find the manifest", err)
			os.Exit(-1)
		}
		repoUrl := fmt.Sprintf("https://%s/v2/%s/manifests/%s", repo, path, digest)
		_, err = callDockerRepo("DELETE", repoUrl, "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)), nil)
		var dockerAuthError *DockerAuthError
		if err != nil && errors.As(err, &dockerAuthError) {
			token, err := retrieveToken(dockerAuthError.AuthRequest, username, password)
			if err != nil {
				log.Panic(err)
			}
			_, err = callDockerRepo("DELETE", repoUrl, "Bearer "+token, nil)
			if err != nil {
				fmt.Printf("%v\n", err)
				os.Exit(-1)
			}
		}

		return nil
	},
}

func getManifestDigest(manifestUrl, username, password string) (string, error) {
	headers := map[string]string{
		"Accept": "application/vnd.oci.image.manifest.v1+json",
	}

	resp, err := callDockerRepo("GET", manifestUrl, "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)), headers)
	var dockerAuthError *DockerAuthError
	if err != nil && errors.As(err, &dockerAuthError) {
		token, err := retrieveToken(dockerAuthError.AuthRequest, username, password)
		if err != nil {
			return "", err
		}
		resp, err = callDockerRepo("GET", manifestUrl, "Bearer "+token, headers)
		if err != nil {
			return "", err
		}

		if len(resp.Errors) > 0 {
			for _, t := range resp.Errors {
				return "", fmt.Errorf("%v", t)
			}
		}

		return resp.DockerContentDigest, nil
	}

	return resp.DockerContentDigest, nil
}
