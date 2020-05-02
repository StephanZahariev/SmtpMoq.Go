package smtpmoq

import (
	"bytes"
	"net/smtp"
	"strconv"
	"sync"
	"testing"

	"github.com/smartystreets/assertions/should"
	"github.com/smartystreets/gunit"
)

func TestExampleFixture(t *testing.T) {
	gunit.Run(new(SimpleTestsFixture), t)
}

type SimpleTestsFixture struct {
	*gunit.Fixture
	server *SMTPServer
}

func (fixture *SimpleTestsFixture) SetupStuff() {
	//Arrange
	var err error
	fixture.server, err = NewServer(":25", "localhost")
	if err != nil {
		fixture.Errorf("Unable to start the SmtpMoq server %v", err)
	}
}
func (fixture *SimpleTestsFixture) TeardownStuff() {
	fixture.server.Stop()
}

func (fixture *SimpleTestsFixture) TestSimpleEmailSend() {
	//Act
	err := sendEmail("sender@example.org", "recipient@example.net", "This is the email body.")
	if err != nil {
		fixture.Error(err)
	}

	//Test
	emails := fixture.server.Emails()
	fixture.So(len(emails), should.Equal, 1)
	fixture.So(emails[0].From, should.Equal, "sender@example.org")
	fixture.So(len(emails[0].Recipients), should.Equal, 1)
	fixture.So(emails[0].Recipients[0], should.Equal, "recipient@example.net")
	fixture.So(emails[0].Data, should.ContainSubstring, "This is the email body.")
}

func (fixture *SimpleTestsFixture) TestMultipleEmailSend() {
	//Act
	max := 10

	wg := sync.WaitGroup{}
	for i := 0; i < max; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			err := sendEmail("sender@example.org", "recipient@example.net", "This is the email body."+strconv.Itoa(i))
			if err != nil {
				fixture.Error(err)
			}
		}(i)
	}

	wg.Wait()

	//Test
	emails := fixture.server.Emails()
	fixture.So(len(emails), should.Equal, max)
}

func sendEmail(sender string, recipient string, body string) error {
	c, err := smtp.Dial(":25")
	if err != nil {
		return err
	}
	defer c.Close()

	c.Mail(sender)
	c.Rcpt(recipient)

	wc, err := c.Data()
	if err != nil {
		return err
	}
	defer wc.Close()
	buf := bytes.NewBufferString(body)
	if _, err = buf.WriteTo(wc); err != nil {
		return err
	}
	c.Quit()

	return nil
}
