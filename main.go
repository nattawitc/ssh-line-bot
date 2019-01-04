package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
)

func main() {
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetDefault("port", "8080")

	port := fmt.Sprintf(":%v", viper.GetString("port"))
	lineSecret := viper.GetString("line.secret")
	lineToken := viper.GetString("line.token")
	sshkey := viper.GetString("ssh.key")
	sshHost := viper.GetString("ssh.host") + ":22"
	sshUser := viper.GetString("ssh.user")

	bot, err := linebot.New(lineSecret, lineToken)
	if err != nil {
		fmt.Println("cannot initiate line client:", err)
		return
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey([]byte(sshkey))
	if err != nil {
		fmt.Println("unable to parse private key", err)
		return
	}

	config := &ssh.ClientConfig{
		User: sshUser,
		Auth: []ssh.AuthMethod{
			// Use the PublicKeys method for remote authentication.
			ssh.PublicKeys(signer),
		},
		//HostKeyCallback: ssh.FixedHostKey(hostKey),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Connect to the remote server and perform the SSH handshake.
	client, err := ssh.Dial("tcp", sshHost, config)
	if err != nil {
		log.Fatalf("unable to connect: %v", err)
	}
	defer client.Close()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		events, err := bot.ParseRequest(r)
		if err != nil {
			fmt.Println("cannot parse request:", err)
			return
		}
		for _, e := range events {
			if e.Type == linebot.EventTypeMessage {
				t, ok := e.Message.(*linebot.TextMessage)
				if !ok {
					fmt.Println("cannot convert e.Message to linebot.TextMessage")
					return
				}

				session, err := client.NewSession()
				if err != nil {
					fmt.Println("unable to create session:", err)
					return
				}
				defer session.Close()

				out, _ := session.CombinedOutput(t.Text)
				bot.ReplyMessage(e.ReplyToken,
					linebot.NewTextMessage(string(out)),
				).Do()
			}
		}
	})
	http.ListenAndServe(port, nil)
}
