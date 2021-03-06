package root

import (
    "crypto/rand"
    "crypto/rsa"
    "fmt"
    "io"
    "log"
    "net"
    "os"
    "sync"

    "github.com/spf13/cobra"
    "golang.org/x/crypto/ssh"
)

var port int
var password string
var wideopen bool
var should_repeat bool

var rootCmd = &cobra.Command{
    Use:   "sshcat",
    Short: "sshcat",
    Long:  "sshcat - Ad-hoc pipe plumbing over SSH",
    Run: func(cmd *cobra.Command, args []string) {
        if ! wideopen && len(password) == 0 {
            log.Println("Error: must provide a password or specify `--wideopen'")
            log.Print(cmd.UsageString())
            os.Exit(1)
        }

        start()
    },
}

func Execute() {
    if err := rootCmd.Execute(); err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
}

func init() {
    rootCmd.Flags().IntVarP(&port, "port", "p", 2222, "port number to listen on")
    rootCmd.Flags().StringVar(&password, "password", "", "")
    rootCmd.Flags().BoolVar(&wideopen, "wideopen", false, "allow unauthenticated connections from anywhere")
    rootCmd.Flags().BoolVarP(&should_repeat, "loop", "k", false, "listen for further connections")
}

func start() {
    var accept_connections = true

    listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
    if err != nil {
        log.Fatalf("Failed to listen on port %d: %s", port, err)
    }
    defer listener.Close()

    for accept_connections {
        tcpConn, err := listener.Accept()
        if err != nil {
            log.Fatalf("Failed to accept connection (%s)", err)
        }

        if ! should_repeat {
            listener.Close()
        }

        handleConnection(tcpConn)

        accept_connections = should_repeat
    }
}

func handleConnection(tcpConn net.Conn) {
    defer tcpConn.Close()

    config := &ssh.ServerConfig{
        ServerVersion: "SSH-2.0-sshcat",
        PasswordCallback: checkPassword,
        NoClientAuth: wideopen,
    }

    hostKey, err := generateHostKey()
    if err != nil {
        log.Fatalf("Failed to generate host key (%s)", err)
    }
    config.AddHostKey(hostKey)

    // Take our TCP connection and upgrade it to an SSH connection.  An SSH
    // connection is actually quite a complicated multiplexed thing, consisting
    // of a series of 'channels' (not to be confused with Go channels) and
    // 'requests'.  The Go SSH library presents both of these as (Go) channels
    // for us to handle, since they can arrive at any time during the session.
    _, chans, reqs, err := ssh.NewServerConn(tcpConn, config)
    if err != nil {
        log.Printf("SSH handshake failed: %s", err)
        return
    }

    // We don't need to do anything with any SSH requests for our simple
    // use-case, but if all requests and SSH channels are not serviced in some
    // way the connection will hang.
    go ssh.DiscardRequests(reqs)

    // The first channel is the one that you'd normally think of as 'the SSH
    // connection', and is what we're going to use
    go handleChannel(<-chans)

    // There shouldn't be any further requests, but just in case, listen for
    // and reject them.
    for newChannel := range chans {
        newChannel.Reject(ssh.Prohibited, "There can be only one")
    }
}

func handleChannel(newChannel ssh.NewChannel) {
    var pipes sync.WaitGroup

    if t := newChannel.ChannelType(); t != "session" {
        newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("Unknown channel type: %s", t))
        log.Fatal("Initial SSH channel was not a session")
    }

    // Like the main SSH connection, each SSH channel _also_ comes with a (Go)
    // channel of requests in addition to the connection itself.
    conn, reqs, err := newChannel.Accept()
    if err != nil {
        log.Fatal("Could not accept SSH channel: %s", err)
    }

    // We can mostly ignore these out-of-band requests, but, again like the
    // main session request, they need to be servied to prevent hangs.
    // We also _do_ need to listen for and respond to the 'shell' request,
    // which is the SSH client asking for a login shell.  We tell it it's going
    // to get one...
    go func() {
        for req := range reqs {
            switch req.Type {
            case "shell":
                // We only accept the default shell
                // (i.e. no command in the Payload)
                if len(req.Payload) == 0 {
                    req.Reply(true, nil)
                }
            default:
                if req.WantReply {
                    req.Reply(false, nil)
                }
            }
        }
    }()

    // ... but then just plumb up the connection to our own STDIO.
    pipes.Add(2)
    go func() {
        io.Copy(conn, os.Stdin)
        pipes.Done()
    }()

    go func() {
        io.Copy(os.Stdout, conn)
        pipes.Done()
    }()

    pipes.Wait()
    conn.Close()
}

func generateHostKey() (ssh.Signer, error) {
    key, err := rsa.GenerateKey(rand.Reader, 2048)
    if err != nil {
        return nil, err
    }
    return ssh.NewSignerFromKey(key)
}

func checkPassword(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
    if string(pass) == password {
        return nil, nil
    }
    return nil, fmt.Errorf("Invalid password")
}
