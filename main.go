package main

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/alexflint/go-arg"
)

// port range: 8163-8180

// ideas:
// https://github.com/liujianping/job

// alternates:
// https://github.com/cosiner/flag

type commandMerge struct {
	Port int `default:"8163"`
}

type commandSplit struct {
	Port int `default:"8164"`
}

type commandExec struct {
	Timeout time.Duration `arg:"-t"`
	Command string        `arg:"positional,required"`
	Args    []string      `arg:"positional"`
}

func main() {
	var args struct {
		Merge *commandMerge `arg:"subcommand:merge"`
		Split *commandSplit `arg:"subcommand:split"`
		Exec  *commandExec  `arg:"subcommand:exec"`
	}

	p := arg.MustParse(&args)
	if p.Subcommand() == nil {
		p.Fail("missing subcommand")
	}

	if args.Merge != nil {
		doMerge(args.Merge)
	}

	if args.Split != nil {
		doSplit(args.Split)
	}

	if args.Exec != nil {
		doExec(args.Exec)
		fmt.Println("done done")
	}
}

func doMerge(args *commandMerge) {
	s, err := net.Listen("tcp", fmt.Sprintf(":%d", args.Port))
	if err != nil {
		panic(err)
	}

	handle := func(conn net.Conn) {
		defer func() { _ = conn.Close() }()
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			panic(err)
		}
	}

	for {
		conn, err := s.Accept()
		if err != nil {
			panic(err)
		}
		go handle(conn)
	}
}

func NewRoom() *Room {
	return &Room{members: make(map[net.Conn]chan string)}
}

type Room struct {
	members map[net.Conn]chan string
	sync.Mutex
	closed bool
}

func (r *Room) Join(conn net.Conn) <-chan string {
	r.Lock()
	defer r.Unlock()

	if r.closed {
		panic("join on closed room")
	}

	c := make(chan string)
	r.members[conn] = c
	return c
}

func (r *Room) Leave(conn net.Conn) {
	r.Lock()
	defer r.Unlock()

	// TODO: return err if conn isn't a member
	c := r.members[conn]
	close(c)
	delete(r.members, conn)
}

func (r *Room) Send(msg string) {
	r.Lock()
	defer r.Unlock()

	if r.closed {
		panic("send on closed room")
	}

	for _, c := range r.members {
		c <- msg
	}
}

func (r *Room) Close() {
	r.Lock()
	defer r.Unlock()

	if r.closed {
		panic("close on closed room")
	}

	r.closed = true
	for conn, c := range r.members {
		close(c)
		delete(r.members, conn)
	}
}

func doSplit(args *commandSplit) {
	r := NewRoom()

	s, err := net.Listen("tcp", fmt.Sprintf(":%d", args.Port))
	if err != nil {
		panic(err)
	}

	// main loop to read from stdin
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			r.Send(scanner.Text())
		}
		if err := scanner.Err(); err != nil {
			panic(err)
		}
		_ = s.Close()
		r.Close()
	}()

	handle := func(conn net.Conn) {
		c := r.Join(conn)

		defer func() {
			r.Leave(conn)
			_ = conn.Close()
		}()

		for {
			msg := <-c
			_, err := fmt.Fprintf(conn, "%s\n", msg)
			if err != nil {
				break
			}
		}
	}

	for {
		conn, err := s.Accept()
		if err != nil {
			break
		}
		go handle(conn)
	}
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

func execCommand(command string, args []string) *exec.Cmd {
	cmd := exec.Command(command, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func doExec(args *commandExec) {
	if args.Timeout > 0 {
		// fmt.Println("will timeout")
	}

	p := parseLines(os.Stdin)
	ticker := time.NewTicker(args.Timeout)

	cmd := execCommand(args.Command, args.Args)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(err)
	}

	if err := cmd.Start(); err != nil {
		panic(err)
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
				panic(err)
			}

			cmd = execCommand(args.Command, args.Args)
			stdin, err = cmd.StdinPipe()
			if err != nil {
				panic(err)
			}

			if err = cmd.Start(); err != nil {
				panic(err)
			}
		}
	}
	// fmt.Println("stdin done")

	if err := cmd.Wait(); err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			os.Exit(err.ExitCode())
		}
		panic(err)
	}

	fmt.Println("out", os.Getpid())
}
