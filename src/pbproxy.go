package main

import (
	"fmt"
	"os"
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
		// if handle, err := pcap.OpenLive(inter_face, 1600, true, pcap.BlockForever); err != nil {
		// 	panic(err)
		// } else if err := handle.SetBPFFilter(expr_string); err != nil { // BPF
		// 	panic(err)
		// } else {
		// 	defer handle.Close()
		// 	// fmt.Println("Listening on " + inter_face + " [" + expr_string + "]")
		// 	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
		// 	for packet := range packetSource.Packets() {
		// 		packetData := packet.Data()
		// 		time := packet.Metadata().Timestamp
		// 		detectDNSSpoof(packetData, time)
		// 	}
		// }
	} else {
		// mode = "client"
		fmt.Println("Client mode")
		var input string
		for {
			fmt.Scanln(&input)
			// fmt.Println(input)
		}
	}

}
