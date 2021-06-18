package exec

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

type Command struct {
	Timeout time.Duration `arg:"-t"`
	Command string        `arg:"positional,required"`
	Args    []string      `arg:"positional"`
}

func (args *Command) Do() error {
	if args.Timeout > 0 {
		// fmt.Println("will timeout")
	}

	p := parseLines(os.Stdin)
	ticker := time.NewTicker(args.Timeout)

	cmd := execCommand(args.Command, args.Args)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

read_stdin:
	for {
		select {
		case line, ok := <-p:
			if !ok {
				// looks like our stdin closed
				// fmt.Println("not ok")
				_ = stdin.Close()
				break read_stdin
			}
			_, err := fmt.Fprintf(stdin, "%s\n", line)
			if err != nil {
				// looks like the subprocess stdin closed
				_ = os.Stdin.Close()
				break read_stdin
			}

		case <-ticker.C:
			// fmt.Println("Tick at", t)
			stdin.Close()
			if err := cmd.Wait(); err != nil {
				return err
			}

			cmd = execCommand(args.Command, args.Args)
			stdin, err = cmd.StdinPipe()
			if err != nil {
				return err
			}

			if err = cmd.Start(); err != nil {
				return err
			}
		}
	}
	// fmt.Println("stdin done")

	if err := cmd.Wait(); err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			os.Exit(err.ExitCode())
		}
		return err
	}

	fmt.Println("out", os.Getpid())

	return nil
}

func execCommand(command string, args []string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func parseLines(reader io.Reader) <-chan string {
	c := make(chan string)
	go func() {
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			c <- scanner.Text()
		}
		close(c)
	}()
	return c
}
