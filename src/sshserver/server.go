package sshserver

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/viper"
	"github.com/vrajashkr/sshellkeeper/src/identity"
	"golang.org/x/crypto/ssh"
)

type SSHServer struct {
	idp  identity.IdentityProvider
	host string
	port string
	cfg  *ssh.ServerConfig
}

func NewSSHServer(idp identity.IdentityProvider, host string, port string) (SSHServer, error) {
	server := SSHServer{
		idp:  idp,
		host: host,
		port: port,
	}

	err := server.initSSHServerConfig()
	if err != nil {
		return SSHServer{}, err
	}

	return server, nil
}

func (sh *SSHServer) initSSHServerConfig() error {
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
			username := conn.User()
			_, err := sh.idp.IsValidUser(username)
			if err != nil {
				return nil, err
			}
			return &ssh.Permissions{Extensions: map[string]string{"username": username}}, nil
		},
	}

	hostKeyPath := viper.GetString("server.ssh_host_key_file")

	privateKeyBytes, err := os.ReadFile(hostKeyPath)
	if err != nil {
		slog.Error("failed to load pkey bytes", "error", err.Error())
		return err
	}

	privateKey, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		slog.Error("failed to parse pkey", "error", err.Error())
		return err
	}

	config.AddHostKey(privateKey)

	sh.cfg = config

	return nil
}

func WriteLinesToChan(ch ssh.Channel, lines []string) error {
	data := strings.Join(lines, "\r\n") + "\r\n"

	slog.Debug("requested to write data to channel", "data", data)
	n, err := ch.Write([]byte(data))
	if err != nil {
		return err
	}
	slog.Debug("wrote bytes to channel", "num", n)

	return nil
}

func writeCharToChan(ch ssh.Channel, b byte) error {
	slog.Debug("requested to write byte to channel", "data", b)
	n, err := ch.Write([]byte{b})
	if err != nil {
		return err
	}
	slog.Debug("wrote byte to channel", "num", n)

	return nil
}

func ReadDataFromChannel(logger slog.Logger, ch ssh.Channel) (string, error) {
	buf := make([]byte, 0)
	tempBuf := make([]byte, 1)

	for {
		n, err := ch.Read(tempBuf)
		logger.Debug("read bytes from channel", "num", n)
		if err != nil {
			logger.Error("failed to read from channel", "error", err.Error())
			break
		}

		// exit for Enter key and Ctrl+C
		if tempBuf[0] == 13 || tempBuf[0] == 3 {
			err = WriteLinesToChan(ch, []string{""})
			if err != nil {
				logger.Error("failed to write empty line to channel", "error", err.Error())
			}
			break
		} else {
			buf = append(buf, tempBuf...)
			err = writeCharToChan(ch, tempBuf[0])
			if err != nil {
				logger.Error("failed to write byte to channel", "error", err.Error())
			}
		}

		logger.Debug("read char from channel", "char", tempBuf[0])
	}

	answer := string(buf)
	return answer, nil
}

func handleServerConn(username string, chans <-chan ssh.NewChannel, handler func(string, ssh.Channel)) {
	for newChan := range chans {
		if newChan.ChannelType() != "session" {
			_ = newChan.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		ch, _, err := newChan.Accept()
		if err != nil {
			slog.Error("Error accepting channel: " + err.Error())
			continue
		}

		handler(username, ch)
		ch.Close()
	}
}

func (sh *SSHServer) Listen(handler func(string, ssh.Channel)) {
	listener, err := net.Listen("tcp", fmt.Sprintf("%s:%s", sh.host, sh.port))
	if err != nil {
		slog.Error("failed to start SSH listener", "error", err.Error())
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			slog.Error("failed to accept incoming connection", "error", err.Error())
			continue
		}

		go func() {
			slog.Debug("start handshake for", "addr", conn.RemoteAddr().String())
			serverConn, chans, reqs, err := ssh.NewServerConn(conn, sh.cfg)
			if err != nil {
				if err == io.EOF || errors.Is(err, syscall.ECONNRESET) {
					slog.Debug("handshake terminated", "error", err.Error())
				} else {
					slog.Error("handshake error", "error", err.Error())
				}
				return
			}

			slog.Debug("connection received", "addr", serverConn.RemoteAddr().String(), "clientVersion", string(serverConn.ClientVersion()))

			go ssh.DiscardRequests(reqs)
			go handleServerConn(serverConn.Permissions.Extensions["username"], chans, handler)
		}()
	}
}
