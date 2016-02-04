package main

import (
	"fmt"
	"github.com/flosch/pongo2"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
)

type Hooks struct {
	Name string
}

func renderHook(hookName string, m Machine, config Config) (string, error) {

	hookName = path.Join(config.HookPath, hookName)
	if _, err := os.Stat(hookName); err != nil {
		log.Println(fmt.Sprintf("%s hook does not exist", hookName))
		return "", err
	}

	var tpl = pongo2.Must(pongo2.FromFile(hookName))
	result, err := tpl.Execute(pongo2.Context{"machine": m, "config": config})
	if err != nil {
		log.Println(fmt.Sprintf("Cannot render hook: %s ", hookName))
		return "", err
	}
	return result, err
}

func executeHooks(hookType string, m Machine, config Config) error {

	var hooks []string
	if hookType == "pre-hook" {
		hooks = config.PreHooks
	} else {
		hooks = config.PostHooks
	}

	for _, hookName := range hooks {
		result, err := renderHook(hookName, m, config)
		if err != nil {
			log.Println(fmt.Sprintf("Something went wrong"))
			return err
		}
		tempFile, err := generateTempFile(hookName, result)
		if err != nil {
			log.Println(fmt.Sprintf("Something went wrong"))
			return err
		}

		err = executeFile(tempFile)
		if err != nil {
			log.Println(fmt.Sprintf("Cannot execute %s", tempFile))
			return err
		}
	}
	return nil
}

func generateTempFile(hookName string, renderedHook string) (filename string, err error) {
	tmpDir := "/tmp/"
	filename = path.Join(tmpDir, hookName)
	f, err := os.Create(filename)
	if err != nil {
		fmt.Println(err)
	}
	n, err := io.WriteString(f, renderedHook)
	if err != nil {
		fmt.Println(n, err)
	}
	f.Close()

	err = os.Chmod(filename, 0700)
	if err != nil {
		fmt.Println(err)
	}

	return filename, err
}

func deleteTempFile(filename string) error {
	err := os.Remove(filename)
	if err != nil {
		fmt.Println(err)
	}
	return err
}

func executeFile(cmd string) error {
	if err := exec.Command(cmd).Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	log.Println(fmt.Sprintf("Sucessfully executed %s.", cmd))
	if err := deleteTempFile(cmd); err != nil {
		fmt.Println("Cannot delete temporary hook file.")
		return err
	}
	return nil
}
