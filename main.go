package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

const header = `
PASS {pass}
NICK {nick}
USER {nick} {nick} gophers.irc.slack.com :User Name
`

const timeout = 20 * time.Second

func setDeadline(conn net.Conn) {
	conn.SetDeadline(time.Now().Add(timeout))
}

func main() {
	if len(os.Args) < 3 {
		log.Fatalf("Usage: slackbug <nick> <IRC password token>")
	}
	nick := os.Args[1]
	pass := os.Args[2]
	login := strings.Replace(header, "{pass}", pass, -1)
	login = strings.Replace(login, "{nick}", nick, -1)
	login = strings.TrimLeft(strings.Replace(login, "\n", "\r\n", -1), "\r\n")

	for i := 1; i < 10; i++ {
		test(login)
	}
}

func test(login string) {
	log.Printf("Connecting...")
	conn, err := tls.Dial("tcp", "gophers.irc.slack.com:6667", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	s := bufio.NewScanner(conn)

	log.Printf("Sending identification...")
	write(conn, login)

	log.Printf("Waiting for welcome message...")
	expect(s, func(line string) bool {
		return command(line) == "001"
	})

	go func() {
		time.Sleep(3 * time.Second)
		log.Printf("Sending PING and waiting for PONG...")
		write(conn, "PING :foo\r\n")
	}()

	expect(s, func(line string) bool {
		return command(line) == "PONG" && strings.HasSuffix(line, ":foo")
	})

	log.Printf("Test was successful.")
}

func write(conn net.Conn, lines string) {
	s := bufio.NewScanner(bytes.NewBufferString(lines))
	for s.Scan() {
		line := strings.Replace(s.Text(), os.Args[2], "XXXXXXXXXX", -1)
		log.Printf("Writing: %s", line)
	}
	setDeadline(conn)
	_, err := conn.Write([]byte(lines))
	if err != nil {
		log.Fatal(err)
	}
}

func expect(s *bufio.Scanner, condition func(line string) bool) {
	for s.Scan() {
		log.Printf("Reading: %s", s.Text())
		if condition(s.Text()) {
			return
		}
	}
	if s.Err() != nil {
		log.Fatal(s.Err())
	}
	log.Fatal("EOF")
}

func command(line string) string {
	fields := strings.Fields(line)
	if len(fields) > 0 && strings.HasPrefix(fields[0], ":") {
		fields = fields[1:]
	}
	if len(fields) > 0 {
		return fields[0]
	}
	return ""
}
