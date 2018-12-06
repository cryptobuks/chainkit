package node

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/blocklayerhq/chainkit/project"
	"github.com/blocklayerhq/chainkit/ui"
	"github.com/blocklayerhq/chainkit/util"
)

func initialize(ctx context.Context, p *project.Project) error {
	_, err := os.Stat(p.StateDir())

	// Skip initialization if already initialized.
	if err == nil {
		return nil
	}

	// Make sure we got an ErrNotExist - fail otherwise.
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	ui.Info("Generating configuration and genesis files")
	if err := util.DockerRun(ctx, p, "init"); err != nil {
		//NOTE: some cosmos app (e.g. Gaia) take a --moniker option in the init command
		// if the normal init fail, rerun with `--moniker $(hostname)`
		hostname, err := os.Hostname()
		if err != nil {
			return err
		}
		if err := util.DockerRun(ctx, p, "init", "--moniker", hostname); err != nil {
			return err
		}
	}

	if err := ui.Tree(p.StateDir(), nil); err != nil {
		return err
	}

	return nil
}

// updateConfig updates the config file for the node before starting.
func updateConfig(file string, vars map[string]string) error {
	config, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	output := bytes.NewBufferString("")
	scanner := bufio.NewScanner(bytes.NewReader(config))
	for scanner.Scan() {
		line := scanner.Text()
		// Scan vars to replace in the current line
		for k, v := range vars {
			if strings.HasPrefix(line+" = ", k) {
				line = fmt.Sprintf("%s = %s", k, v)
			}
		}
		if _, err := fmt.Fprintln(output, line); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	f, err := os.OpenFile(file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	if _, err := io.Copy(f, output); err != nil {
		return err
	}

	return nil
}