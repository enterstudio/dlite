package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/satori/go.uuid"
	"github.com/tsuru/config"
	"github.com/urfave/cli"
)

var initCommand = cli.Command{
	Name:        "init",
	Usage:       "initialize your dlite installation",
	Description: "perform setup of dlite for the currently logged in user",
	Action: func(ctx *cli.Context) error {
		currentUser := getUser()
		configPath := getPath(currentUser)
		binPath := filepath.Join(configPath, "bin")
		configFile := filepath.Join(configPath, "config.yaml")
		diskFile := filepath.Join(configPath, "disk.qcow")
		cfg := Config{}

		err := config.ReadConfigFile(configFile)
		if err == nil {
			fmt.Println("WARNING: It appears you have already initialized dlite. Continuing will destroy your current virtual machine and its configuration.")
			if !confirm("Continue? (y/n)") {
				return cli.NewExitError("Aborting initialization...", 1)
			}
		}

		fmt.Println("")

		err = os.RemoveAll(configPath)
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}

		err = os.MkdirAll(configPath, 0755)
		if err != nil {
			return cli.NewExitError(err.Error(), 1)
		}

		cfg.Id = uuid.NewV1().String()

		cfg.Hostname = askString("Virtual machine hostname", "local.docker")
		cfg.Disk = askInt("Disk size (in gigabytes)", 20)
		cfg.Cpu = askInt("CPU cores to allocate to VM", 2)
		cfg.Memory = askInt("Memory to allocate to VM (in gigabytes)", 2)

		host, _ := getHostAddress()
		cfg.DNS = askString("DNS server", host)
		cfg.Docker = askString("Docker version", "latest")
		cfg.Extra = ask("Extra flags to pass to the docker daemon")
		cfg.Route = askBool("Allow direct connections to containers", true)

		fmt.Println("")

		if err := spin("Saving configuration", func() error {
			return writeConfig(configPath, cfg)
		}); err.ExitCode() != 0 {
			return err
		}

		if err := spin("Creating ssh key pair", func() error {
			return generateKeys(currentUser)
		}); err.ExitCode() != 0 {
			return err
		}

		if err := spin("Adding host to ssh config", func() error {
			return addSSHConfig(currentUser, cfg.Hostname)
		}); err.ExitCode() != 0 {
			return err
		}

		if err := spin("Creating tool binaries", func() error {
			err := os.MkdirAll(binPath, 0755)
			if err != nil {
				return err
			}

			for _, tool := range []string{"com.docker.hyperkit", "qcow-tool"} {
				bin, err := Asset(tool)
				if err != nil {
					return err
				}
				err = ioutil.WriteFile(filepath.Join(binPath, tool), bin, 0755)
				if err != nil {
					return err
				}
			}

			return nil
		}); err.ExitCode() != 0 {
			return err
		}

		if err := spin("Creating disk", func() error {
			return buildDisk(filepath.Join(binPath, "qcow-tool"), diskFile, cfg.Disk, currentUser.Uid, currentUser.Gid)
		}); err.ExitCode() != 0 {
			return err
		}

		if err := spin("Downloading OS", func() error {
			return downloadOS(configPath)
		}); err.ExitCode() != 0 {
			return err
		}

		fmt.Println("")
		fmt.Println("Next we'll run a few steps that require sudo, you may be prompted for your password.")
		return runSetup(cfg.Hostname, currentUser.Home)
	},
}
