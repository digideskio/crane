// Copyright 2016 crane authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/tsuru/tsuru/cmd"
	"gopkg.in/yaml.v1"
)

type serviceCreate struct{}

func (c *serviceCreate) Info() *cmd.Info {
	desc := "Creates a service based on a passed manifest. The manifest format should be a yaml and follow the standard described in the documentation (should link to it here)"
	return &cmd.Info{
		Name:    "create",
		Usage:   "create path/to/manifest [- for stdin]",
		Desc:    desc,
		MinArgs: 1,
	}
}

type serviceYaml struct {
	Id       string
	Username string
	Password string
	Endpoint map[string]string
	Team     string
}

func (c *serviceCreate) Run(context *cmd.Context, client *cmd.Client) error {
	manifest := context.Args[0]
	u, err := cmd.GetURL("/services")
	if err != nil {
		return err
	}
	var data []byte
	if manifest == "-" {
		data, err = ioutil.ReadAll(os.Stdin)
	} else {
		data, err = ioutil.ReadFile(manifest)
	}
	if err != nil {
		return err
	}
	var y serviceYaml
	err = yaml.Unmarshal(data, &y)
	if err != nil {
		return err
	}
	v := url.Values{}
	v.Set("id", y.Id)
	v.Set("password", y.Password)
	v.Set("username", y.Username)
	v.Set("team", y.Team)
	v.Set("endpoint", y.Endpoint["production"])
	request, err := http.NewRequest("POST", u, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "Service successfully created")
	return nil
}

type serviceRemove struct {
	cmd.ConfirmationCommand
}

func (c *serviceRemove) Run(context *cmd.Context, client *cmd.Client) error {
	serviceName := context.Args[0]
	question := fmt.Sprintf("Are you sure you want to remove the service %q?", serviceName)
	if !c.Confirm(context, question) {
		return nil
	}
	url, err := cmd.GetURL("/services/" + serviceName)
	if err != nil {
		return err
	}
	request, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintln(context.Stdout, "Service successfully removed.")
	return nil
}

func (c *serviceRemove) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "remove",
		Usage:   "remove <servicename>",
		Desc:    "removes a service from catalog",
		MinArgs: 1,
	}
}

type serviceList struct{}

func (c *serviceList) Info() *cmd.Info {
	return &cmd.Info{
		Name:  "list",
		Usage: "list",
		Desc:  "list services that belongs to user's team and it's service instances.",
	}
}

func (c *serviceList) Run(ctx *cmd.Context, client *cmd.Client) error {
	url, err := cmd.GetURL("/services")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	b, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	rslt, err := cmd.ShowServicesInstancesList(b)
	if err != nil {
		return err
	}
	ctx.Stdout.Write(rslt)
	return nil
}

type serviceUpdate struct{}

func (c *serviceUpdate) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "update",
		Usage:   "update <path/to/manifest>",
		Desc:    "Update service data, extracting it from the given manifest file.",
		MinArgs: 1,
	}
}

func (c *serviceUpdate) Run(ctx *cmd.Context, client *cmd.Client) error {
	manifest := ctx.Args[0]
	b, err := ioutil.ReadFile(manifest)
	if err != nil {
		return err
	}
	var y serviceYaml
	err = yaml.Unmarshal(b, &y)
	if err != nil {
		return err
	}
	v := url.Values{}
	v.Set("id", y.Id)
	v.Set("password", y.Password)
	v.Set("username", y.Username)
	v.Set("team", y.Team)
	v.Set("endpoint", y.Endpoint["production"])
	u, err := cmd.GetURL(fmt.Sprintf("/services/%s", y.Id))
	if err != nil {
		return err
	}
	request, err := http.NewRequest("PUT", u, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(request)
	if err != nil {
		return err
	}
	if resp.StatusCode == http.StatusOK {
		fmt.Fprintln(ctx.Stdout, "Service successfully updated.")
	}
	return nil
}

type serviceDocAdd struct{}

func (c *serviceDocAdd) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "doc-add",
		Usage:   "service doc-add <service> <path/to/docfile>",
		Desc:    "Update service documentation, extracting it from the given file.",
		MinArgs: 2,
	}
}

func (c *serviceDocAdd) Run(ctx *cmd.Context, client *cmd.Client) error {
	serviceName := ctx.Args[0]
	u, err := cmd.GetURL("/services/" + serviceName + "/doc")
	if err != nil {
		return err
	}
	docPath := ctx.Args[1]
	b, err := ioutil.ReadFile(docPath)
	v := url.Values{}
	v.Set("doc", string(b))
	request, err := http.NewRequest("PUT", u, strings.NewReader(v.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	fmt.Fprintf(ctx.Stdout, "Documentation for '%s' successfully updated.\n", serviceName)
	return nil
}

type serviceDocGet struct{}

func (c *serviceDocGet) Run(ctx *cmd.Context, client *cmd.Client) error {
	serviceName := ctx.Args[0]
	url, err := cmd.GetURL("/services/" + serviceName + "/doc")
	if err != nil {
		return err
	}
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}

	b, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	ctx.Stdout.Write(b)
	return nil
}

func (c *serviceDocGet) Info() *cmd.Info {
	return &cmd.Info{
		Name:    "doc-get",
		Usage:   "service doc-get <service>",
		Desc:    "Shows service documentation.",
		MinArgs: 1,
	}
}

type serviceTemplate struct{}

func (c *serviceTemplate) Info() *cmd.Info {
	usg := `template
e.g.: $ crane template`
	return &cmd.Info{
		Name:  "template",
		Usage: usg,
		Desc:  "Generates a manifest template file and places it in current directory",
	}
}

const passwordSize = 12

func generatePassword() (string, error) {
	b := make([]byte, passwordSize)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func (c *serviceTemplate) Run(ctx *cmd.Context, client *cmd.Client) error {
	pass, err := generatePassword()
	if err != nil {
		return err
	}
	template := `id: servicename
username: username_to_auth
password: %s
team: team_responsible_to_provide_service
endpoint:
  production: production-endpoint.com`
	template = fmt.Sprintf(template, pass)
	f, err := os.Create("manifest.yaml")
	defer f.Close()
	if err != nil {
		return errors.New("Error while creating manifest template.\nOriginal error message is: " + err.Error())
	}
	f.Write([]byte(template))
	fmt.Fprintln(ctx.Stdout, `Generated file "manifest.yaml" in current directory`)
	return nil
}
