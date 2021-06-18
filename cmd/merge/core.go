package merge

import (
	"bufio"
	"fmt"
	"net"
)

type Command struct {
	Port int `default:"8163"`
}

func (args *Command) Do() error {
	s, err := net.Listen("tcp", fmt.Sprintf(":%d", args.Port))
	if err != nil {
		return err
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
			return err
		}
		go handle(conn)
	}

	return nil
}
