//+build mage

package main

import (
	"os"
	"os/exec"
)

func run(prog string, args ...string) {
	cmd := exec.Command(prog, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}

func ClearCache() {
	run("bash", "-c", "free && sync && echo 3 | sudo tee /proc/sys/vm/drop_caches && free")
}
