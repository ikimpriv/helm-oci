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

type tagsT struct {
	cli.Helper
	PrintName bool `cli:"print-name" usage:"Print manifest name before tags"`
}

func (argv *tagsT) Validate(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return fmt.Errorf("usage: helm-repo tags <repository/path>")
	}
	repoParts := strings.SplitN(ctx.Args()[0], "/", 2)
	if len(repoParts) < 2 {
		return fmt.Errorf("usage: helm-repo tags <repository/path>")
	}
	return nil
}

var tags = &cli.Command{
	Name: "tags",
	Desc: "List artifact tags",
	Argv: func() interface{} { return new(tagsT) },
	Fn: func(ctx *cli.Context) error {
		argv := ctx.Argv().(*tagsT)
		spec := ctx.Args()[0]

		spec = strings.TrimPrefix(spec, "oci://")

		repoParts := strings.SplitN(spec, "/", 2)
		repo := repoParts[0]
		path := repoParts[1]

		username, password, err := getCredentials(repo)
		if err != nil {
			log.Printf("Error: %v\n", err)
			os.Exit(-1)
		}

		repoUrl := fmt.Sprintf("https://%s/v2/%s/tags/list", repo, path)
		resp, err := callDockerRepo("GET", repoUrl, "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)), nil)
		var dockerAuthError *DockerAuthError
		if err != nil && errors.As(err, &dockerAuthError) {
			token, err := retrieveToken(dockerAuthError.AuthRequest, username, password)
			if err != nil {
				log.Panic(err)
			}
			resp, err = callDockerRepo("GET", repoUrl, "Bearer "+token, nil)
			if err != nil {
				var dockerError DockerError
				if errors.As(err, &dockerError) {
					if dockerError.Code == "NAME_UNKNOWN" {
						fmt.Printf("Manifest not found: %s\n", path)
					} else {
						fmt.Printf("Error: %#v\n", dockerError)
					}
					os.Exit(-1)
				} else {
					log.Panic(err)
				}
			}

			if argv.PrintName {
				fmt.Println(path)
			}
			for _, t := range resp.Tags {
				if argv.PrintName {
					fmt.Print("  ")
				}
				fmt.Println(t)
			}
		}

		return nil
	},
}
