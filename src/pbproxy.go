package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/pbkdf2"
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
	pwd, err := ioutil.ReadFile(pwdfile)
	if err != nil {
		panic(err)
	}
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
		key := pbkdf2.Key([]byte(pwd), []byte("tempsalt"), 4096, 32, sha1.New)
		nonce, _ := hex.DecodeString("64a9433eae7ccceee2fc0eda")
		block, err := aes.NewCipher(key)
		if err != nil {
			panic(err.Error())
		}

		aesgcm, err := cipher.NewGCM(block)
		if err != nil {
			panic(err.Error())
		}

		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}

		for {
			netData, err := bufio.NewReader(c).ReadString('\n')
			ciphertext := []byte(netData)
			plaintext, err := aesgcm.Open(nil, nonce, ciphertext, nil)
			netData = fmt.Sprintf("%s", plaintext)
			if err != nil {
				panic(err.Error())
			}
			if err != nil {
				// fmt.Println(err)
				fmt.Println("A user disconnected. Continue to listen for other user...")
				c, err = l.Accept()
				continue
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
			fmt.Println("Error: ", err)
			return
		}
		key := pbkdf2.Key([]byte(pwd), []byte("tempsalt"), 4096, 32, sha1.New)

		// https://pkg.go.dev/crypto/cipher#example-NewGCM-Encrypt
		block, err := aes.NewCipher(key)
		if err != nil {
			panic(err.Error())
		}
		// Never use more than 2^32 random nonces with a given key because of the risk of a repeat.
		// nonce := make([]byte, 12)
		// if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		// 	panic(err.Error())
		// }
		nonce, _ := hex.DecodeString("64a9433eae7ccceee2fc0eda")

		aesgcm, err := cipher.NewGCM(block)
		if err != nil {
			panic(err.Error())
		}

		for {
			reader := bufio.NewReader(os.Stdin)
			fmt.Print(">> ")
			plaintext, _ := reader.ReadString('\n')
			plaintext += "\n"
			ciphertext := aesgcm.Seal(nil, nonce, []byte(plaintext), nil)
			fmt.Fprintf(c, string(ciphertext))

			message, _ := bufio.NewReader(c).ReadString('\n')
			fmt.Print("->: " + message)
			if strings.TrimSpace(string(plaintext)) == "STOP" {
				fmt.Println("TCP client exiting...")
				return
			}
		}
	}

}
