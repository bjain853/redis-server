package main

import (
	"fmt"
	"net"
	"strings"
)

const defaultPort = ":5000"

type Config struct {
	Port string
}

type Server struct {
	cfg Config
	ln  net.Listener
}

func NewServer(cfg Config, listener net.Listener) *Server {
	if len(cfg.Port) == 0 {
		cfg.Port = defaultPort
	}

	return &Server{
		cfg,
		listener,
	}
}

func main() {
	var PORT = fmt.Sprintf(":%d", 6379)
	fmt.Printf("Listening on port %s", PORT)
	listener, err := net.Listen("tcp", PORT)

	if err != nil {
		fmt.Println(err)
		return
	}

	aof, err := NewAof("database.aof")

	if err != nil {
		fmt.Println(err)
		return
	}

	defer aof.Close()

	aof.Read(func(value Value) {
		command := strings.ToUpper(value.array[0].bulk)
		args := value.array[1:]

		handler, ok := Handlers[command]

		if !ok {
			fmt.Println("Invalid command ", command)
			return
		}
		handler(args)
	})

	connection, err := listener.Accept()
	if err != nil {
		fmt.Println(err)
		return
	}

	defer connection.Close()

	for {
		resp := NewResp(connection)
		value, err := resp.Read()
		if err != nil {
			fmt.Println(err)
			return
		}
		if value.typ != "array" {
			fmt.Println("Invalid request, expected an array")
			continue
		}
		if len(value.array) == 0 {
			fmt.Println("Invalid request, expected array length > 0")
			continue
		}

		command := strings.ToUpper(value.array[0].bulk)
		args := value.array[1:]

		writer := NewWriter(connection)

		handler, ok := Handlers[command]

		if !ok {
			fmt.Println("Invalid command: ", command)
			writer.Write(Value{typ: "string", str: ""})
			continue
		}

		if command == "SET" || command == "HSET" {
			aof.Write(value)
		}

		result := handler(args)
		writer.Write(result)

	}
}
