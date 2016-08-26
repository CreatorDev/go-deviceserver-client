package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"

	"github.com/urfave/cli"
	ds "gitlab.flowcloud.systems/creator-ops/go-deviceserver-client"
	"gitlab.flowcloud.systems/creator-ops/go-deviceserver-client/hateoas"
)

var createKey = cli.Command{
	Name:      "create-key",
	Aliases:   []string{"ck"},
	Category:  "keys",
	Usage:     "Create a new key/secret",
	ArgsUsage: "<name>",
	Flags:     []cli.Flag{},
	Action: func(c *cli.Context) error {
		keyName := c.Args().Get(0)
		d, err := ds.Create(hateoas.Create(&hateoas.Client{
			EntryURL: deviceserverURL,
		}))
		if err != nil {
			return err
		}
		defer d.Close()

		credentials, err := ReadCredentials()
		if err != nil {
			return err
		}

		err = d.Authenticate(credentials)
		if err != nil {
			return err
		}

		key, err := d.CreateAccessKey(keyName)
		if err != nil {
			return err
		}

		buf, err := json.MarshalIndent(&key, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(buf))

		return nil
	},
}

var listKeys = cli.Command{
	Name:      "list-keys",
	Aliases:   []string{"lk"},
	Category:  "keys",
	Usage:     "Lists the known access keys",
	ArgsUsage: " ",
	Flags:     []cli.Flag{},
	Action: func(c *cli.Context) error {
		d, err := ds.Create(hateoas.Create(&hateoas.Client{
			EntryURL: deviceserverURL,
		}))
		if err != nil {
			return err
		}
		defer d.Close()

		credentials, err := ReadCredentials()
		if err != nil {
			return err
		}

		err = d.Authenticate(credentials)
		if err != nil {
			return err
		}

		keys, err := d.GetAccessKeys()
		if err != nil {
			return err
		}

		for i, key := range keys.Items {
			var selfHref string
			self, err := key.Links.Get("self")
			if err != nil {
				selfHref = "? unable to find self link"
			} else {
				selfHref = self.Href
			}
			fmt.Printf("[%d] '%s' = %s\n  %s\n\n", i, key.Name, key.Key, selfHref)
		}

		return nil
	},
}

var deleteKey = cli.Command{
	Name:      "delete-key",
	Aliases:   []string{"dk"},
	Category:  "keys",
	Usage:     "Delete the specified key",
	ArgsUsage: "<key self URL>",
	Flags:     []cli.Flag{},
	Action: func(c *cli.Context) error {
		self := c.Args().Get(0)

		u, err := url.Parse(self)
		if err != nil {
			return err
		}
		du, err := url.Parse(deviceserverURL)
		if err != nil {
			return err
		}

		if u.Scheme != "http" && u.Scheme != "https" {
			return errors.New("Invalid scheme for self link")
		}
		if u.Host != du.Host {
			return errors.New("self link is not for this deviceserver")
		}

		d, err := ds.Create(hateoas.Create(&hateoas.Client{
			EntryURL: deviceserverURL,
		}))
		if err != nil {
			return err
		}
		defer d.Close()

		credentials, err := ReadCredentials()
		if err != nil {
			return err
		}

		err = d.Authenticate(credentials)
		if err != nil {
			return err
		}

		err = d.Delete(self)
		if err != nil {
			return err
		}

		return nil
	},
}
