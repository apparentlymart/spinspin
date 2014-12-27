package main

import (
	"os/exec"
	"os"
	"time"

	"gopkg.in/fsnotify.v1"
)

func main() {
	if len(os.Args) < 2 {
		os.Stderr.WriteString("Usage: spinspin <cmd> [args...]\n")
		os.Exit(1)
	}

	path := os.Args[1]
	args := os.Args[1:]

	path, err := exec.LookPath(path)
	if err != nil {
		os.Stderr.WriteString("Command not found in path\n")
		os.Exit(1)
	}

	for {
		os.Stdout.WriteString("\033[0m\033[2J\033[0;0H")

		// Unscientifically wait for things to settle before we try to run,
		// to try to work around issues where the program is still busy
		// after we are notified of a write.
		time.Sleep(500 * time.Millisecond)

		cmd := new(exec.Cmd)
		cmd.Path = path
		cmd.Args = args
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			os.Stderr.WriteString("Failed to watch executable\n")
			os.Exit(1)
		}
		watcher.Add(path)

		running := false
		err = cmd.Start()
		if err == nil {
			running = true
		} else {
			os.Stdout.WriteString("\033[31m--- failed to start ---\033[0m\n")
		}

		restart := make(chan bool)
		go func () {
			for {
				select {
				case event := <-watcher.Events:
					if event.Op & fsnotify.Write == fsnotify.Write {
						if running {
							cmd.Process.Kill()
						}
						restart <- true
					}
				case <-watcher.Errors:
					os.Stderr.WriteString("Watcher error. Restarting.\n")
					if running {
						cmd.Process.Kill()
					}
					restart <- true
				}
			}
		}()

		if running {
			err = cmd.Wait()
			running = false
			if err == nil {
				os.Stdout.WriteString("\n\033[1;30m--- exited successfully ---\033[0m\n")
			} else {
				os.Stdout.WriteString("\n\033[31m--- exited with error ---\033[0m\n")
			}
		}

		// Wait for the restart signal
		<-restart

		watcher.Close()

	}
}
