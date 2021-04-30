package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
	// "Crypto"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("", r)
		}
	}()
	// fmt.Println("Hello world!")
	argv := os.Args[1:]      // Argument vector
	argv_length := len(argv) // Length of the arguments

	// fmt.Printf("Argument vector: %v\n", argv)
	// fmt.Printf("Vector Length: %v\n", argv_length)

	var (
		listenport  string = "-1"
		pwdfile     string = "-1"
		optind      int    = 0
		destination string = "-1"
		port        string = "-1"
	)
	if argv_length != 4 && argv_length != 6 {
		panic("Incorrect input length!")
	}

	opt := argv[optind]
	if opt == "-l" {
		listenport = argv[optind+1]
		optind += 2
	} else if opt == "-p" {
		pwdfile = argv[optind+1]
	} else {
		panic("Unknown flag passed into pbroxy!")
	}
	opt = argv[optind]
	if optind == 2 && opt == "-p" {
		pwdfile = argv[optind+1]
	} else if optind == 2 && opt != "-p" {
		panic("Incorrect Input! Try again...")
	}
	optind += 2
	destination = argv[optind]
	port = argv[optind+1]
	fmt.Println(pwdfile, destination, port)
	if listenport != "-1" {
		// mode = "reverse"
		fmt.Println("Reverse-proxy mode")
		PORT := ":" + listenport
		l, err := net.Listen("tcp", PORT)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer l.Close()

		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}

		for {
			netData, err := bufio.NewReader(c).ReadString('\n')
			if err != nil {
				fmt.Println(err)
				return
			}
			if strings.TrimSpace(string(netData)) == "STOP" {
				fmt.Println("Exiting TCP server!")
				return
			}

			fmt.Print("-> ", string(netData))
			t := time.Now()
			myTime := t.Format(time.RFC3339) + "\n"
			c.Write([]byte(myTime))
		}
	} else {
		// mode = "client"
		fmt.Println("Client mode")
		// var input string
		// for {
		// 	fmt.Scanln(&input)
		// 	// fmt.Println(input)
		// }
		// The TCP server and client setup is comming from the below website
		// https://www.linode.com/docs/guides/developing-udp-and-tcp-clients-and-servers-in-go/

		destIpPort := destination + ":" + port
		c, err := net.Dial("tcp", destIpPort)
		if err != nil {
			fmt.Println(err)
			return
		}

		for {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print(">> ")
			text, _ := reader.ReadString('\n')
			fmt.Fprintf(c, text+"\n")

			message, _ := bufio.NewReader(c).ReadString('\n')
			fmt.Print("->: " + message)
			if strings.TrimSpace(string(text)) == "STOP" {
				fmt.Println("TCP client exiting...")
				return
			}
		}
	}

}
