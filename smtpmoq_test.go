package smtpmoq

import (
	"bytes"
	"log"
	"net/smtp"
	"testing"
)

func TestSendingSimpleEmail(t *testing.T) {
	//Arange
	server := SMTPServer{}
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			t.Fatalf("Unable to start the SmtpMoq server %v", err)
		}
	}()

	//Act
	c, err := smtp.Dial(":25")
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	c.Mail("sender@example.org")
	c.Rcpt("recipient@example.net")

	wc, err := c.Data()
	if err != nil {
		t.Fatal(err)
	}
	defer wc.Close()
	buf := bytes.NewBufferString("This is the email body.")
	if _, err = buf.WriteTo(wc); err != nil {
		t.Fatal(err)
	}
	c.Quit()

	//Test
	if len(server.Emails) != 1 {
		t.Errorf("len(server.Emails) = %v; want 1", len(server.Emails))
	}
	if server.Emails[0].From != "sender@example.org" {
		t.Errorf("Sender = %s; want sender@example.org", server.Emails[0].From)
	}
	if len(server.Emails[0].Recipients) != 1 {
		t.Errorf("len(server.Emails[0].Recipients) = %v; want 1", len(server.Emails[0].Recipients))
	}
	if server.Emails[0].Recipients[0] != "recipient@example.net" {
		t.Errorf("Recipient = %s; want recipient@example.net", server.Emails[0].From)
	}
	if server.Emails[0].Data != "This is the email body." {
		t.Errorf("server.Emails[0].Data = %s; want This is the email body.", server.Emails[0].From)
	}
}
