package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"gopkg.in/urfave/cli.v2"
	"gopkg.in/yaml.v2"
)

const (
	detectFailureSnippet = `if [ $? -ne 0 ]; then
	run_on_failure
	exit 1
fi`
)

type BuildSpec struct {
	BeforeInstall []string `yaml:"before-install"`
	Install       []string `yaml:"install"`
	BeforeScript  []string `yaml:"before_script"`
	Script        []string `yaml:"script"`
	AfterSuccess  []string `yaml:"after_success"`
	AfterFailure  []string `yaml:"after_failure"`
}

func main() {
	app := &cli.App{
		Name:  "travis-build",
		Usage: "Convert a travis build script into a shell script",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Specify a file path to write output other wise output is written to stdout",
			},
		},
		Action: func(c *cli.Context) error {
			if c.Args().Len() != 1 {
				return fmt.Errorf("No path for input file provided")
			}

			scriptPath := c.Args().Get(0)
			if _, err := os.Stat(scriptPath); err != nil {
				return fmt.Errorf("Provided input file '%s' does not exist", scriptPath)
			}

			bytes, err := ioutil.ReadFile(scriptPath)
			if err != nil {
				return err
			}

			var spec BuildSpec
			err = yaml.Unmarshal(bytes, &spec)
			if err != nil {
				return err
			}

			var w io.Writer = os.Stdout
			outputPath := c.String("output")
			if len(outputPath) > 0 {
				f, err := os.Create(outputPath)
				if err != nil {
					return err
				}
				bufferedWriter := bufio.NewWriter(f)
				defer bufferedWriter.Flush()
				w = bufferedWriter
			}

			fmt.Fprintf(w, "#!/bin/sh -x\n")
			fmt.Fprintf(w, "# NOTE: This script is automatically generated. DO NOT TOUCH!!\n\n")

			var steps []string
			steps = append(steps, spec.BeforeInstall...)
			steps = append(steps, spec.Install...)
			steps = append(steps, spec.BeforeScript...)
			steps = append(steps, spec.Script...)

			fmt.Fprintf(w, "run_on_failure() {\n")
			if len(spec.AfterFailure) > 0 {
				for _, step := range spec.AfterFailure {
					fmt.Fprintf(w, "\t%s\n", step)
				}
			} else {
				fmt.Fprintf(w, "\techo \"Nothing to run after failed build\"\n")
			}
			fmt.Fprintf(w, "}\n\n")

			for _, step := range steps {
				fmt.Fprintf(w, "%s\n", step)
				fmt.Fprintf(w, "%s\n\n", detectFailureSnippet)
			}

			for _, step := range spec.AfterSuccess {
				fmt.Fprintf(w, "%s\n", step)
			}

			return nil
		},
	}

	app.Run(os.Args)
}
