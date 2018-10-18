package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os/exec"
	"path"
	"strings"

	"golang.org/x/crypto/ssh"
)


var reposLocation = "/tmp/repos"

func main() {
	port := 2221

	config := &ssh.ServerConfig{
		//Explicitely set "none" auth as valid. This is wanted to allow anonymous SSH
		NoClientAuth: true,
	}

	privateBytes, err := ioutil.ReadFile("./id_rsa")
	if err != nil {
		log.Fatal("Failed to load private key (./id_rsa)")
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key")
	}

	config.AddHostKey(private)

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", port))
	if err != nil {
		log.Fatalf("Failed to listen on 2200 (%s)", err)
	}

	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept incoming connection (%s)", err)
			continue
		}
		// Before use, a handshake must be performed on the incoming net.Conn.
		sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
		if err != nil {
			log.Printf("Failed to handshake (%s)", err)
			continue
		}

		log.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())
		// Discard all global out-of-band Requests
		go ssh.DiscardRequests(reqs)
		// Accept all channels but we only really want "x"
		go handleChannels(chans)
	}
}

func handleChannels(chans <-chan ssh.NewChannel) {
	// Service the incoming Channel channel in go routine
	for newChannel := range chans {
		go handleChannel(newChannel)
	}
}
func handleChannel(newChannel ssh.NewChannel) {

	if t := newChannel.ChannelType(); t != "session" {
		newChannel.Reject(ssh.Prohibited, "neeeeeerp")
		return
	}

	channel, requests, err := newChannel.Accept()
	if err != nil {
		log.Printf("Could not accept channel (%s)", err)
		return
	}

	go func() {
		for req := range requests {

			// only going to respond to exec.
			// and then only to git-upload-pack
			if req.Type == "exec" {
				if req.WantReply {
					req.Reply(true, nil)
				}
				fmt.Println("exec")
				fmt.Printf("%x\n", req.Payload)

				// strip the first 2 bytes
				// Only want the git-upload-pack
				parts := strings.Split(string(req.Payload[4:]), " ")

				// we only expect git-upload-pack repo.git
				if len(parts) != 2 {
					req.Reply(false, nil)
					return
				}

				if 0 == strings.Compare(parts[0], "git-upload-pack") {

					// ensure supplied path does contain dir traversal
					p := path.Clean(parts[1])

					// get rid of enclosing ''
					p = strings.Trim(p, "'")

					fullPath := path.Join(reposLocation, p)
					log.Printf("Requesting repo: %s\n", fullPath)

					cmd := exec.Command("/usr/bin/git-upload-pack", "--strict", fullPath)
					cmd.Stdin = channel
					cmd.Stdout = channel
					cmd.Stderr = channel.Stderr()
					cmd.Run()
					channel.SendRequest("exit-status", false, nil)

				} else {
					req.Reply(false, nil)
					return
				}

			} else {
				if req.WantReply {
					req.Reply(false, nil)
				}
			}
		}
	}()
}
