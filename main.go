package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os/exec"
	"path"
	"strings"

	"golang.org/x/crypto/ssh"
)

var reposLocation string

func main() {
	portPtr := flag.Int("p", 2221, "Port to use")
	dirPtr := flag.String("d", "./repos", "The directory where the git repositories are")
	hostkeyPtr := flag.String("s", "./id_rsa", "Where to find the host-key")

	flag.Parse()

	reposLocation = *dirPtr

	config := &ssh.ServerConfig{
		//Explicitely set "none" auth as valid. This is wanted to allow anonymous SSH
		NoClientAuth: true,
	}

	privateBytes, err := ioutil.ReadFile(*hostkeyPtr)
	if err != nil {
		log.Printf("If you need to generate a host key, use: ssh-keygen -t rsa")
		log.Fatalf("Failed to load private key %s (%q)", *hostkeyPtr, err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatalf("Failed to parse private key: %q", err)
	}

	config.AddHostKey(private)

	listener, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%d", portPtr))
	if err != nil {
		log.Fatalf("Failed to listen on %d (%s)", portPtr, err)
	}

	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept incoming connection (%q)", err)
			continue
		}
		// Before use, a handshake must be performed on the incoming net.Conn.
		sshConn, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
		if err != nil {
			log.Printf("Failed to handshake (%s)", err)
			continue
		}

		log.Printf("New SSH connection from %s (%s)", sshConn.RemoteAddr(), sshConn.ClientVersion())

		// Accept all channels but we only really want "x"
		go handleConnection(reqs, chans)
	}
}

func handleConnection(reqs <-chan *ssh.Request, chans <-chan ssh.NewChannel) {

	go func(reqs <-chan *ssh.Request) {
		for r := range reqs {
			if r.WantReply {
				r.Reply(false, nil)
			}
		}
	}(reqs)

	// Service the incoming Channel channel in go routine
	for ch := range chans {
		if t := ch.ChannelType(); t == "session" {

			channel, requests, err := ch.Accept()
			if err != nil {
				log.Printf("Could not accept channel (%s)", err)
				return
			}
			go handleChannel(channel, requests)
		} else {
			ch.Reject(ssh.Prohibited, "neeeeeerp")
		}
	}
}
func handleChannel(channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()

	for req := range requests {

		// only going to respond to exec.
		// and then only to git-upload-pack
		if req.Type == "exec" {
			if req.WantReply {
				req.Reply(true, []byte{})
			}

			go func() {
				handleExecChannel(channel, req)
				channel.Close()
			}()
		} else {
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}

}

type execRequestMsg struct {
	Command string
}

type exitStatusMsg struct {
	Status uint32
}

func handleExecChannel(channel ssh.Channel, req *ssh.Request) {
	fmt.Println("exec")

	var msg execRequestMsg
	ssh.Unmarshal(req.Payload, &msg)

	// Only want the git-upload-pack
	parts := strings.Split(msg.Command, " ")

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

		res := doExec(channel, channel, channel.Stderr(), fullPath)

		channel.SendRequest("exit-status", false, ssh.Marshal(&exitStatusMsg{res}))
	}
}

func doExec(stdin io.Reader, stdout, stderr io.Writer, path string) uint32 {

	cmd := exec.Command("/usr/bin/git-upload-pack", "--strict", path)
	cmd.Dir = path
	cmd.Env = []string{fmt.Sprintf("GIT_DIR=%s", path)}
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	if err := cmd.Run(); err != nil {
		log.Printf("Error occurred: %q\n", err)
		return 1
	}
	return 0
}
