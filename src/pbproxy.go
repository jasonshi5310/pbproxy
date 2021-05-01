package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"regexp"

	"golang.org/x/crypto/pbkdf2"
	"golang.org/x/crypto/sha3"
)

const SaltSize = 16
const MaxUint = ^uint(0)
const MaxInt = int(MaxUint >> 50)

// https://www.gregorygaines.com/blog/posts/2020/6/11/how-to-hash-and-salt-passwords-in-golang-using-sha512-and-why-you-shouldnt#3
// Generate 16 bytes randomly and securely using the
// Cryptographically secure pseudorandom number generator (CSPRNG)
// in the crypto.rand package
func generateRandomSalt() []byte {
	var salt = make([]byte, SaltSize)

	_, err := rand.Read(salt[:])

	if err != nil {
		panic(err)
	}
	return salt
}

// https://github.com/vfedoroff/go-netcat/blob/87e3e79d77ee6a0b236a784be83759a4d002a20d/main.go#L16
// read stream from src and write to dst
func stream_copy(src io.Reader, dst io.Writer, pwd string, encrypt bool) <-chan int {
	buf := make([]byte, MaxInt)
	sync_channel := make(chan int)
	go func() {
		defer func() {
			if con, ok := dst.(net.Conn); ok {
				con.Close()
				log.Printf("Connection from %v is closed\n", con.RemoteAddr())
			}
			if r := recover(); r != nil {
				fmt.Println(r)
			}
			sync_channel <- 0 // Notify that processing is finished
		}()
		for {
			var nBytes int
			var err error
			nBytes, err = src.Read(buf)
			if err != nil {
				// if err != io.EOF {
				// 	log.Printf("Read error: %s\n", err)
				// }
				break
			}
			actualBuf := make([]byte, nBytes)
			actualBuf = append(actualBuf[:0], buf[:nBytes]...)
			var resultBuf []byte
			//https://tutorialedge.net/golang/go-encrypt-decrypt-aes-tutorial/
			if encrypt {
				salt := generateRandomSalt() // Genrate random salt
				// Generate new 32 bytes key for AES256 using pbkdf2
				key := pbkdf2.Key([]byte(pwd), salt, 4096, 32, sha3.New512)
				// fmt.Println("KEY: ", key)
				c, err := aes.NewCipher(key)
				// if there are any errors, handle them
				if err != nil {
					fmt.Println("Error generating new AES Cipher", err)
					return
				}
				gcm, err := cipher.NewGCM(c)
				// if any error generating new GCM
				// handle them
				if err != nil {
					fmt.Println("Error generating new GCM", err)
					return
				}

				// creates a new byte array the size of the nonce
				// which must be passed to Seal
				nonce := make([]byte, gcm.NonceSize())
				// populates our nonce with a cryptographically secure
				// random sequence
				if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
					fmt.Println(err)
					return
				}
				saltAndNonce := make([]byte, SaltSize+gcm.NonceSize())
				saltAndNonce = append(saltAndNonce[:0], salt[:]...)
				saltAndNonce = append(saltAndNonce[:], nonce[:]...)
				// here we encrypt our text using the Seal function
				// Seal encrypts and authenticates plaintext, authenticates the
				// additional data and appends the result to dst, returning the updated
				// slice. The nonce must be NonceSize() bytes long and unique for all
				// time, for a given key.
				resultBuf = gcm.Seal(saltAndNonce, nonce, actualBuf, nil)

			} else {
				// Decrypt key
				salt := actualBuf[:SaltSize]
				nonceAndCiphertext := actualBuf[SaltSize:]
				key := pbkdf2.Key([]byte(pwd), salt, 4096, 32, sha3.New512)
				c, err := aes.NewCipher(key)
				if err != nil {
					fmt.Println(err)
				}

				gcm, err := cipher.NewGCM(c)
				if err != nil {
					fmt.Println(err)
				}

				nonceSize := gcm.NonceSize()
				if len(nonceAndCiphertext) < nonceSize {
					fmt.Println(err)
				}
				nonce := nonceAndCiphertext[:nonceSize]
				ciphertext := nonceAndCiphertext[nonceSize:]
				resultBuf, err = gcm.Open(nil, nonce, ciphertext, nil)
				if err != nil {
					fmt.Println(err)
				}
			}
			nBytes = len(resultBuf)
			_, err = dst.Write(resultBuf[0:nBytes])
			if err != nil {
				log.Fatalf("Write error: %s\n", err)
			}
		}
	}()
	return sync_channel
}

// https://dev.to/alicewilliamstech/getting-started-with-sockets-in-golang-2j66
// Hangle each conncetion concurently
func handleConnection(handleCilentConn net.Conn, desIP string, desPort string, pwd string) {

	conToSsh, err := net.Dial("tcp4", desIP+":"+desPort)
	if err != nil {
		msg := "Cannot find this server: " + desIP + ":" + desPort
		handleCilentConn.Write([]byte(msg))
		handleCilentConn.Close()
		return
	}

	chan_to_ssh := stream_copy(handleCilentConn, conToSsh, pwd, false)
	chan_to_client := stream_copy(conToSsh, handleCilentConn, pwd, true)
	select {
	case <-chan_to_ssh:
		log.Println("SSH is closed")
	case <-chan_to_client:
		log.Println("Client is terminated")
	}
	handleCilentConn.Close()

}

// https://www.linode.com/docs/guides/developing-udp-and-tcp-clients-and-servers-in-go/#create-a-concurrent-tcp-server
// Concurent server
func reversePorxy(listenPort string, desIP string, desPort string, pwd string) {
	l, err := net.Listen("tcp4", listenPort)
	if err != nil {
		fmt.Println("Listen err: ", err)
		return
	}
	defer l.Close()

	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println("Error connecting:", err.Error())
			return
		}
		log.Println("Client " + c.RemoteAddr().String() + " connected.")

		go handleConnection(c, desIP, desPort, pwd)
	}
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println(r)
		}
	}()
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
		panic("Incorrect input length! Try again as the following:\n\tgo run pbproxy.go [-l listenport] -p pwdfile destination port")
	}

	var dashP = regexp.MustCompile(`^-p$`)
	var dashL = regexp.MustCompile(`^-l$`)
	var anyFlag = regexp.MustCompile(`^-.*`)
	for i := 0; i < 2; i++ {
		opt := argv[optind]
		switch {
		case dashP.MatchString(opt):
			{
				if pwdfile != "-1" {
					panic("Multiple pwdfile provided!")
				}
				pwdfile = argv[optind+1]
				optind += 2

			}
		case dashL.MatchString(opt):
			{
				if listenport != "-1" {
					panic("Multiple listenport provided!")
				}
				listenport = argv[optind+1]
				optind += 2
			}
		case anyFlag.MatchString(opt):
			{
				panic("Unrecognize Flag provided! Try again as the following:\n\tgo run pbproxy.go [-l listenport] -p pwdfile destination port")
			}
		}
	}
	if optind+1 >= argv_length {
		panic("Good try! Please input valid values other than -1!")
	}
	destination = argv[optind]
	port = argv[optind+1]
	pwd, err := ioutil.ReadFile(pwdfile)
	if err != nil {
		fmt.Print("PWDFILE: ", pwdfile)
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
		// The TCP client setup is comming from the below website
		// https://www.linode.com/docs/guides/developing-udp-and-tcp-clients-and-servers-in-go/
		destIpPort := destination + ":" + port
		clientConn, err := net.Dial("tcp4", destIpPort)
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}
		localSsh_to_reverse := stream_copy(os.Stdin, clientConn, string(pwd), true)
		reverse_to_client := stream_copy(clientConn, os.Stdout, string(pwd), false)
		select {
		case <-localSsh_to_reverse:
			// log.Println("Local SSH is closed")
		case <-reverse_to_client:
			// log.Println("Local Client is terminated")
		}
	}

}
