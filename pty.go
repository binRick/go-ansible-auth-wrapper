package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/k0kubun/pp"
	"github.com/kr/pty"
	"golang.org/x/term"
)

func ReadOutput(output chan string, rc io.ReadCloser) {
	r := bufio.NewReader(rc)
	for {
		x, _ := r.ReadString('\n')
		output <- string(x)
	}
}

var read_stdin_bytes = []string{}
var read_stdout_bytes = []string{}

func stdout_writer_logic() {
	//	var buf = make([]byte, bufSz)
	scanner := bufio.NewScanner(cmd_ptmx)
	ssh_pass_prompt := false
	for scanner.Scan() {
		t := scanner.Text()
		if false {
			l.Printf("\n\nstdout_reader_logic>> %s\n", t)
		}
		read_stdout_bytes = append(read_stdout_bytes, string(t))
		if false {
			l.Printf("      ssh pass prompt?      %v\n", ssh_pass_prompt)
		}
	}
	//	_, _ = io.Copy(os.Stdout, cmd_ptmx)
	//nr, _ := cmd_ptmx.Read(buf)
	//		os.Stdout.Write(buf[:nr])
	//		l.Printf("stdout_reader_logic %d bytes >> %s\n", nr, string(buf[:nr]))

}

func stdin_reader_logic() {
	var buf = make([]byte, bufSz)
	for {
		nr, _ := os.Stdin.Read(buf)
		cmd_ptmx.Write(buf[:nr])
		read_stdin_bytes = append(read_stdin_bytes, string(buf[:nr]))
		if false {
			l.Printf("stdin_reader_logic %d bytes >> %s\n", nr, string(buf[:nr]))
			l.Printf("\n%s\n", read_stdout_bytes)
		}
	}
	/*
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			fmt.Println(`stdin_reader_logic>>`, scanner.Text())
		}*/
}

var (
	cmd_ptmx = &(os.File{})
)

func exec_pty_io_logic() error {
	ptmx, err := pty.Start(exec.Command("bash"))
	f(err)
	pp.Println(ptmx)
	cmd_ptmx = ptmx
	defer func() {
		_ = cmd_ptmx.Close()
		l.Printf("pty closed......... | read_stdin_bytes %d bytes | read_stdout_bytes %d bytes | \n", len(read_stdin_bytes), len(read_stdout_bytes))
		l.Printf("\n%s\n", read_stdout_bytes)
		l.Printf("\n%s\n", read_stdin_bytes)
	}()

	//fmt.Fprintln(cmd_ptmx, `date|tee -a /tmp/kk`)
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, cmd_ptmx); err != nil {
				log.Printf("error resizing pty: %s", err)
			}
		}
	}()
	ch <- syscall.SIGWINCH                        // Initial resize.
	defer func() { signal.Stop(ch); close(ch) }() // Cleanup signals when done.

	// Set stdin in raw mode.
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	f(err)
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }() // Best effort.
	go stdout_writer_logic()
	go stdin_reader_logic()
	fmt.Fprintln(cmd_ptmx, sub_command)
	fmt.Fprintln(cmd_ptmx, `exit`)

	// Copy stdin to the pty and the pty to stdout.
	// NOTE: The goroutine will keep reading until the next keystroke before returning.
	go func() {
		//		_, _ = io.Copy(cmd_ptmx, os.Stdin)
	}()
	cb, _ := io.Copy(os.Stdout, cmd_ptmx)
	pp.Println(cb, `copied bytes`)

	return nil
}

//buffer := bytes.Buffer{}
//buffer.Write([]byte("username\npassword\n"))
