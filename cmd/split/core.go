package split

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sync"
)

type Command struct {
	Port int `default:"8164"`
}

func (args *Command) Do() error {
	r := NewRoom()

	s, err := net.Listen("tcp", fmt.Sprintf(":%d", args.Port))
	if err != nil {
		return err
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

	return nil
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
