package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/google/goterm/term"
	"github.com/k0kubun/pp"
	//"github.com/runletapp/go-console"
)

var ( //   Common
	DEBUG_MODE               = true
	PTY_ROWS                 = 120
	PTY_COLS                 = 60
	l                        = log.New(os.Stderr, "", 0) //  Debug logger to this process stderr
	this_program_binary_name = os.Args[0]
)

var ( //		Parsed prefix and suffix commands
	sub_command      = ``
	wrapper_command  = ``
	sub_command_exec = `` //		Absolute path to sub command
)

var ( //		Common strings
	sub_command_missing_error_msg_prefix       = fmt.Sprintf(`No Sub Command defined. You should prefix this program to your ansible invocation.`)
	ansible_example_cmd_suffix                 = fmt.Sprintf(`%s`, `-i inventory.txt -u user -k`)
	ansible_example_cmd_sudo_suffix            = fmt.Sprintf(`%s`, `-bK`)
	ansible_example_cmd_playbook_suffix        = fmt.Sprintf(`%s`, `path/to/playbook.yaml`)
	cmd_seperator_msg                          = fmt.Sprintf(`%s`, "This is an optional parameter which is used to clearly seperate the prefix and suffix commands, facilitating both commands to accept arguments.")
	cmd_seperator                              = fmt.Sprintf(`%s`, `--`)
	sub_command_exit_code                      = -1
	sub_command_ok                             = false
	sub_command_stdout_read_bytes              = []byte{}
	sub_command_stdout_lines                   = []string{}
	sub_command_started                        = time.Now()
	sub_command_pid                            = -1
	sub_command_user_cpu_time            int64 = 0
	sub_command_dur                            = time.Duration(-1 * time.Second)
)

var ( // 		Add ansi colors
	cmd_seperator_formatted                        = term.Cyanf(`%s`, cmd_seperator)
	sub_command_missing_error_msg_prefix_formatted = term.BBlackf(`%s`, term.Redf(`%s`, term.Underline(sub_command_missing_error_msg_prefix)))
	cmd_seperator_msg_formatted                    = term.Cyanf(`%s`, cmd_seperator_msg)
)

const (
	cmd_file = "/tmp/ansible-auth-wrapper-"   // file the base filename of the logfile
	bufSz    = 8192                           // BUFSZ size of the buffers used for IO
	Welcome  = "Examplescript up and running" // Welcome Welcome message printed when the application starts
	//   Error & Debug Template
	sub_command_missing_error_msg = `%s
===================
%s
===================
  %s %s ansible-playbook %s %s
  %s %s ansible-playbook %s %s

===================
%s
===================
  %s %s ansible host.com %s %s -m ping
  %s ansible host.com %s -m ping

%s

===================
Debug:
===================
| Debug Mode?                      %v
| Command Prefix:                  %s
| Command Suffix:                  %s
| Suffix Command Exec:             %s
| Sub Command Exit Code:           %d
| Sub Command OK?                  %v
| Sub Command PID:                 %d
| Sub Command Bytes Read:          %d
| Sub Command Stdout Lines Qty:    %d
| Sub Command Execution Duration:  %s
`
)

func init() {
	validate_sub_command()
	wrapper_command, sub_command = extract_commands()
	validate_sub_cmd_path()
	if DEBUG_MODE {
		fmt.Print(gen_error_msg())
		//os.Exit(0)
	}
}

func exec_pty_io_logic() error {
	pty, _ := term.OpenPTY()
	defer pty.Close()
	backupTerm, _ := term.Attr(os.Stdin)
	myTerm := backupTerm
	myTerm.Raw()
	myTerm.Set(os.Stdin)
	backupTerm.Set(pty.Slave)
	defer backupTerm.Set(os.Stdin)
	go Snoop(pty)
	sig := make(chan os.Signal, 2)
	signal.Notify(sig, syscall.SIGWINCH, syscall.SIGCLD)
	cmd := exec.Command(os.Getenv("SHELL"), "")
	//pp.Println(sub_command)
	//cmd := exec.Command(sub_command)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = pty.Slave, pty.Slave, pty.Slave
	cmd.Args = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true}
	cmd.Start()
	myTerm.Winsz(os.Stdin)
	myTerm.Winsz(pty.Slave)
	for {
		switch <-sig {
		case syscall.SIGWINCH:
			myTerm.Winsz(os.Stdin)
			myTerm.Setwinsz(pty.Slave)
		default:
			return nil
		}
	}
	return nil
}

func Snoop(pty *term.PTY) {
	pid := os.Getpid()
	pidcol, _ := term.NewColor256(strconv.Itoa(pid), strconv.Itoa(pid%256), "")
	msg := fmt.Sprint("\n", term.Green(Welcome), " pid:", pidcol, " file:", term.Yellow(cmd_file+strconv.Itoa(pid)+"\n"))
	fmt.Println(msg)
	F, err := os.Create(cmd_file + strconv.Itoa(pid))
	f(err)
	cmd_file_fd, err := os.OpenFile(cmd_file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	f(err)

	datawriter := bufio.NewWriter(cmd_file_fd)

	for _, data := range []string{fmt.Sprintf(`bash -c "%s"`, sub_command), `exit`} {
		_, _ = datawriter.WriteString(data + "\n")
	}

	go reader(pty.Master, F)
	go writer(pty.Master)

	datawriter.Flush()
	//	cmd_file_fd.Close()
}

func reader(master *os.File, L *os.File) {
	var buf = make([]byte, bufSz)
	defer func() {
		L.Sync()
		L.Close()
	}()
	for {
		nr, _ := master.Read(buf)
		os.Stdout.Write(buf[:nr])
		L.Write(buf[:nr])
	}
}

// writer reads from stdin and writes to master
func writer(master *os.File) {
	var buf = make([]byte, bufSz)
	for {
		nr, _ := os.Stdin.Read(buf)
		master.Write(buf[:nr])
	}
}

func main() {
	pp.Println(os.Args)
	pp.Println(`extract_sub_command=>`, sub_command)
	msg := fmt.Sprintf(`pty dev
cmd:       %s
	`,
		sub_command,
	)
	l.Println(term.Blue(msg))
	if false {

		//	f(wgpty())
	} else {
		f(exec_pty_io_logic())
	}
	fmt.Print(gen_error_msg())

}

func gen_error_msg() string {
	return fmt.Sprintf(sub_command_missing_error_msg,
		sub_command_missing_error_msg_prefix_formatted,
		term.Yellow(`Ansible Playbook Usage Examples:`),
		term.Yellow(this_program_binary_name), cmd_seperator_formatted, ansible_example_cmd_suffix, ansible_example_cmd_playbook_suffix,
		term.Yellow(this_program_binary_name), ansible_example_cmd_suffix, ansible_example_cmd_sudo_suffix, ansible_example_cmd_playbook_suffix,
		term.Yellow(`Ansible Usage Examples:`),
		term.Yellow(this_program_binary_name), cmd_seperator_formatted, ansible_example_cmd_suffix, ansible_example_cmd_sudo_suffix,
		term.Yellow(this_program_binary_name), ansible_example_cmd_suffix,
		cmd_seperator_msg_formatted,
		DEBUG_MODE, wrapper_command, sub_command, sub_command_exec, sub_command_exit_code, sub_command_ok, sub_command_pid, len(sub_command_stdout_read_bytes), len(sub_command_stdout_lines), sub_command_dur,
	)
}

func do_err() {
	fmt.Print(gen_error_msg())
	os.Exit(1)
}

func validate_sub_cmd_path() {
	sub_cmd_path, err := exec.LookPath(strings.Split(sub_command, ` `)[0])
	if err != nil || sub_cmd_path == `` {
		do_err()
	}
	sub_command_exec = sub_cmd_path
}

func extract_commands() (string, string) {
	sub_cmd_array := []string{}
	prefix_cmd_array := []string{}
	in_sub_cmd := false
	for _, arg := range os.Args {
		if arg == `--` {
			in_sub_cmd = true
			continue
		}
		if !in_sub_cmd {
			prefix_cmd_array = append(prefix_cmd_array, arg)
		} else {
			sub_cmd_array = append(sub_cmd_array, arg)
		}
	}
	return strings.Join(prefix_cmd_array, ` `), strings.Join(sub_cmd_array, ` `)
}

func validate_sub_command() {
	valid := (len(os.Args) > 1)
	if !valid {
		do_err()
	}
}

/*
func wgpty() error {
	sub_command_started = time.Now()
	proc, err := console.New(PTY_ROWS, PTY_COLS)
	f(err)
	defer proc.Close()
	var args []string

	if runtime.GOOS == "windows" {
		args = []string{"cmd.exe", "/c", sub_command}
	} else {
		args = []string{"bash", "-c", sub_command}
	}

	f(proc.Start(args))

	var wg sync.WaitGroup
	wg.Add(1)
	var copied_bytes int64
	go func() {
		defer wg.Done()
		//sub_command_stdout_read_bytes = append(sub_command_stdout_read_bytes, proc)
		pp.Println("copying!")
		b, err := io.Copy(os.Stdout, proc)
		if err == nil {
			copied_bytes += b
		}
		if copied_bytes > 0 {
			f(err)
		}
	}()

	child, err := proc.Wait()
	if err != nil {
		log.Printf("Wait err: %v\n", err)
	}

	pp.Println(proc.Pid())

	wg.Wait()
	sub_command_exit_code = int(child.ExitCode())
	sub_command_ok = child.Success()
	sub_command_pid, err = proc.Pid()
	f(err)
	sub_command_dur = time.Since(sub_command_started)
	sub_command_user_cpu_time = int64(child.UserTime())
	pp.Println(child.ExitCode())
	pp.Println(child.Exited())
	pp.Println(child.Success())
	pp.Println(child.UserTime())
	return nil
}
*/
func f(err error) {
	if err == nil {
		return
	}
	fmt.Print(gen_error_msg())
	panic(fmt.Sprintf("%s", err))
}
