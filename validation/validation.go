package email

import (
	"crypto/tls"
	"fmt"
	"gopkg.in/gomail.v2"
)

type messageInfo struct {
	body, toAddr string
}

// Allow up to 20 messages to be queued up.
var message = make(chan messageInfo, 20)

func init() {
	go messageDaemon()
}

func messageDaemon() {
	fmt.Println("Started mail daemon")
	d := gomail.NewPlainDialer("localhost", 25, "", "")
	// TODO: Remove this once we have a valid SSL certificate for this domain.
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	for {
		select {
		case m := <-message:
			mess := gomail.NewMessage()
			mess.SetHeader("From", "do-not-reply@5bitstudios.com")
			mess.SetHeader("To", m.toAddr)
			mess.SetHeader("Subject", "Please confirm your email address on eagles-list")
			mess.SetBody("text/html", m.body)
			if err := d.DialAndSend(mess); err != nil {
				fmt.Println(err)
			}
		}
	}
}

// Send a message body to this address.
func SendMessage(body string, toAddr string) {
	message <- messageInfo{body, toAddr}
}
