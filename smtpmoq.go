package smtpmoq

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/google/uuid"
)

const lineBreak = "\r\n"

//SMTPServer provides basic implementation of the Simple Mail Transfer Protocol that holds the emails in memory instead
//of sending them to the recipients. Used during development/integration testing of an app
type SMTPServer struct {
	Addr     string
	Hostname string
	Emails   []EmailMessage
}

//EmailMessage represents an email received by the SMTP server
type EmailMessage struct {
	GUID       uuid.UUID
	From       string
	Recipients []string
	Data       string
}

//ListenAndServe creates a new listener and accepts SMTP traffic
func (server *SMTPServer) ListenAndServe() error {
	addr := server.Addr
	if addr == "" {
		addr = ":25"
	}

	lnr, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer lnr.Close()

	return server.Serve(lnr)
}

//Serve uses an existing listener and accepts SMTP traffic
func (server *SMTPServer) Serve(lnr net.Listener) error {
	for {
		con, err := lnr.Accept()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				log.Printf("smtpd: Accept error: %v", err)
				continue
			}
			return err
		}

		session, err := server.newSession(con)
		if err != nil {
			continue
		}

		go session.serve()
	}
}

func (server *SMTPServer) newSession(con net.Conn) (session *smtpSession, err error) {
	session = &smtpSession{
		server: server,
		con:    con,
		br:     bufio.NewReader(con),
		bw:     bufio.NewWriter(con),
	}

	return
}

type smtpSession struct {
	server *SMTPServer
	con    net.Conn
	br     *bufio.Reader
	bw     *bufio.Writer
}

func (session *smtpSession) serve() {
	defer session.con.Close()

	receivedEmail := EmailMessage{}
	session.sendf("220 %s"+lineBreak, session.server.Hostname)
	for {
		msg, err := session.readMsg()
		if err != nil {
			session.errorf("read error: %v", err)
			return
		}

		cmd := smtpCommand(msg)
		switch cmd.getCmd() {
		case "HELO":
			session.semdMsg("250 %s SmtpMoq server responding", session.server.Hostname)
		case "EHLO":
			session.semdMsg("250 %s SmtpMoq server responding", session.server.Hostname)
			session.semdMsg("250-SMTPUTF8")
		case "NOOP":
			session.send250Ok()
		case "QUIT":
			session.semdMsg("221 It was nice talking to you. Bye.")
			return
		case "RSET":
			receivedEmail = EmailMessage{}
			session.send250Ok()
		case "MAIL FROM":
			receivedEmail.From = extractString(cmd.getCmdPayload(), "<", ">")
			session.send250Ok()
		case "RCPT TO":
			receivedEmail.Recipients = append(receivedEmail.Recipients, extractString(cmd.getCmdPayload(), "<", ">"))
			session.send250Ok()
		case "DATA":
			session.semdMsg("354 Start mail input; end with <CRLF>.<CRLF>")
			for {
				msg, err := session.readMsg()
				if err != nil {
					session.errorf("read error: %v", err)
					return
				}
				if msg == "." {
					break
				}
				receivedEmail.Data = receivedEmail.Data + msg
			}

			receivedEmail.GUID = uuid.New()
			session.server.Emails = append(session.server.Emails, receivedEmail) //TODO: Resolve the race condition
			session.send250OkWithBody("queued as " + receivedEmail.GUID.String())
		case "VRFY":
			session.send250OkWithBody(cmd.getCmdPayload())
		default:
			session.semdMsg("500 Unknow command")
		}
	}
}

func (session *smtpSession) send250Ok() {
	session.send250OkWithBody("")
}

func (session *smtpSession) send250OkWithBody(body string) {
	s := body
	if body != "" {
		s = ": " + s
	}
	session.semdMsg("250 Ok" + s)
}

func (session *smtpSession) sendf(format string, args ...interface{}) {
	fmt.Fprintf(session.bw, format, args...)
	session.bw.Flush()
}

func (session *smtpSession) semdMsg(format string, args ...interface{}) {
	session.sendf(format+lineBreak, args...)
}

func (session *smtpSession) readMsg() (string, error) {
	slice, err := session.br.ReadSlice('\n')
	if err != nil {
		return "", err
	}

	return strings.ReplaceAll(string(slice), lineBreak, ""), nil
}

func (session *smtpSession) errorf(format string, args ...interface{}) {
	log.Printf("Client error: "+format, args...)
}

type smtpCommand string

func (cmd smtpCommand) getCmd() string {
	c := strings.ToUpper(string(cmd))
	if cidx := strings.Index(c, ":"); cidx > 0 {
		return c[:cidx]
	}

	if strings.HasPrefix(c, "HELLO ") {
		return "HELLO"
	} else if strings.HasPrefix(c, "EHLO ") {
		return "EHLO"
	} else if strings.HasPrefix(c, "VRFY ") {
		return "VRFY"
	}

	return c
}

func (cmd smtpCommand) getCmdPayload() string {
	c := cmd.getCmd()
	if len(cmd) == len(c) {
		return ""
	}

	return string(cmd)[len(c)+1:]
}

func extractString(s string, startTag string, endTag string) string {
	startIndex := strings.Index(s, startTag)
	endIndex := strings.Index(s, endTag)
	return s[startIndex+1 : endIndex]
}
