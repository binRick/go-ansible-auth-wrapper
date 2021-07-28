package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/kr/pty"
	"golang.org/x/term"
)

func exec_pty_io_logic() error {
	ptmx, err := pty.Start(exec.Command("bash", `-il`))
	if err != nil {
		return err
	}
	// Make sure to close the pty at the end.
	defer func() {
		_ = ptmx.Close()
		fmt.Println(`pty closed.........`)
	}()

	fmt.Fprintln(ptmx, `date|tee -a /tmp/kk`)
	fmt.Fprintln(ptmx, sub_command)
	fmt.Fprintln(ptmx, `exit`)
	// Handle pty size.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				log.Printf("error resizing pty: %s", err)
			}
		}
	}()
	ch <- syscall.SIGWINCH                        // Initial resize.
	defer func() { signal.Stop(ch); close(ch) }() // Cleanup signals when done.

	// Set stdin in raw mode.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.

	// Copy stdin to the pty and the pty to stdout.
	// NOTE: The goroutine will keep reading until the next keystroke before returning.
	go func() {
		_, _ = io.Copy(ptmx, os.Stdin)
	}()
	_, _ = io.Copy(os.Stdout, ptmx)

	return nil
}

//buffer := bytes.Buffer{}
//buffer.Write([]byte("username\npassword\n"))
