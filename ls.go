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

type lsT struct {
	cli.Helper
	All bool `cli:"all" usage:"Show all nested artifacts"`
}

func (argv *lsT) Validate(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return fmt.Errorf("usage: helm-repo ls <repository/path>")
	}
	return nil
}

var ls = &cli.Command{
	Name: "ls",
	Desc: "List artifacts",
	Argv: func() interface{} { return new(lsT) },
	Fn: func(ctx *cli.Context) error {
		argv := ctx.Argv().(*lsT)
		spec := ctx.Args()[0]

		spec = strings.TrimPrefix(spec, "oci://")

		repoParts := strings.SplitN(spec, "/", 2)
		repo := repoParts[0]

		path := ""
		if len(repoParts) > 1 {
			path = repoParts[1]
		}

		username, password, err := getCredentials(repo)
		if err != nil {
			log.Printf("Error: %v\n", err)
			os.Exit(-1)
		}

		repoUrl := fmt.Sprintf("https://%s/v2/_catalog", repo)

		resp, err := callDockerRepo("GET", repoUrl, "Basic "+base64.StdEncoding.EncodeToString([]byte(username+":"+password)), nil)
		var dockerAuthError *DockerAuthError
		if err != nil && errors.As(err, &dockerAuthError) {
			token, err := retrieveToken(dockerAuthError.AuthRequest, username, password)
			if err != nil {
				log.Panic(err)
			}
			resp, err = callDockerRepo("GET", repoUrl, "Bearer "+token, nil)
			if err != nil {
				fmt.Printf("%v\n", err)
				os.Exit(-1)
			}

			if path != "" && !strings.HasSuffix(path, "/") {
				path = path + "/"
			}

			for _, rep := range resp.Repositories {
				if path == "" || strings.HasPrefix(rep, path) {
					rep = strings.TrimPrefix(rep, path)
					if argv.All || !strings.Contains(rep, "/") {
						fmt.Println(rep)
					}
				}
			}
		}

		return nil
	},
}
