package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
)

// https://github.com/vfedoroff/go-netcat/blob/87e3e79d77ee6a0b236a784be83759a4d002a20d/main.go#L16
func stream_copy(src io.Reader, dst io.Writer) <-chan int {
	buf := make([]byte, 1024)
	sync_channel := make(chan int)
	go func() {
		defer func() {
			if con, ok := dst.(net.Conn); ok {
				con.Close()
				log.Printf("Connection from %v is closed\n", con.RemoteAddr())
			}
			sync_channel <- 0 // Notify that processing is finished
		}()
		for {
			var nBytes int
			var err error
			nBytes, err = src.Read(buf)
			if string(buf) == "logout" || string(buf) == "exit" {
				break
			} else if err != nil {
				// if err != io.EOF {
				// 	log.Printf("Read error: %s\n", err)
				// }
				break
			}
			_, err = dst.Write(buf[0:nBytes])
			if err != nil {
				log.Fatalf("Write error: %s\n", err)
			}
		}
	}()
	return sync_channel
}

// https://dev.to/alicewilliamstech/getting-started-with-sockets-in-golang-2j66
func handleConnection(handleCilentConn net.Conn, desIP string, desPort string, pwd string) {

	conToSsh, err := net.Dial("tcp4", desIP+":"+desPort)
	if err != nil {
		// fmt.Println("Cannot find this server: " + desIP + ":" + desPort)
		msg := "Cannot find this server: " + desIP + ":" + desPort
		handleCilentConn.Write([]byte(msg))
		handleCilentConn.Close()
		return
	}

	chan_to_ssh := stream_copy(handleCilentConn, conToSsh)
	chan_to_client := stream_copy(conToSsh, handleCilentConn)
	select {
	case <-chan_to_ssh:
		log.Println("SSH is closed")
	case <-chan_to_client:
		log.Println("Client is terminated")
	}
	handleCilentConn.Close()

}

// https://www.linode.com/docs/guides/developing-udp-and-tcp-clients-and-servers-in-go/#create-a-concurrent-tcp-server
func reversePorxy(listenPort string, desIP string, desPort string, pwd string) {
	l, err := net.Listen("tcp4", listenPort)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println("Error connecting:", err.Error())
			return
		}
		fmt.Println("Client connected.")

		fmt.Println("Client " + c.RemoteAddr().String() + " connected.")

		go handleConnection(c, desIP, desPort, pwd)
	}
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
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
	pwd, err := ioutil.ReadFile(pwdfile)
	if err != nil {
		panic(err)
	}
	// fmt.Println(string(pwd), destination, port)
	if listenport != "-1" {
		// mode = "reverse"
		fmt.Println("Reverse-proxy mode")
		listenport = ":" + listenport
		reversePorxy(listenport, destination, port, string(pwd))
	} else {
		// mode = "client"
		// fmt.Println("Client mode")
		// The TCP server and client setup is comming from the below website
		// https://www.linode.com/docs/guides/developing-udp-and-tcp-clients-and-servers-in-go/
		destIpPort := destination + ":" + port
		clientConn, err := net.Dial("tcp4", destIpPort)
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}
		localSsh_to_reverse := stream_copy(os.Stdin, clientConn)
		reverse_to_client := stream_copy(clientConn, os.Stdout)
		select {
		case <-localSsh_to_reverse:
			// log.Println("Local SSH is closed")
		case <-reverse_to_client:
			// log.Println("Local Client is terminated")
		}

		// key := pbkdf2.Key([]byte(pwd), []byte("tempsalt"), 4096, 32, sha1.New)

		// // https://pkg.go.dev/crypto/cipher#example-NewGCM-Encrypt
	}

}
