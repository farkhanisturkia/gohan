package mail

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strconv"

	"github.com/farkhanisturkia/gohan/internal/config"
)

func SendEmail(to string, subject string, body string) error {
	env := config.GetEnv()

	if env.MailDriver == "log" || env.MailHost == "" {
		fmt.Printf("[MAIL LOG] To: %s | Subject: %s | Body: %s\n", to, subject, body)
		return nil
	}

	from := env.MailFromAddress
	fromName := env.MailFromName
	if fromName == "" {
		fromName = env.AppName
	}

	headers := make(map[string]string)
	headers["From"] = fmt.Sprintf("%s <%s>", fromName, from)
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	message := ""
	for k, v := range headers {
		message += fmt.Sprintf("%s: %s\r\n", k, v)
	}
	message += "\r\n" + body

	addr := fmt.Sprintf("%s:%s", env.MailHost, env.MailPort)
	host := env.MailHost

	var auth smtp.Auth
	if env.MailUsername != "" {
		auth = smtp.PlainAuth("", env.MailUsername, env.MailPassword, host)
	}

	portInt, _ := strconv.Atoi(env.MailPort)

	if env.MailEncryption == "ssl" || portInt == 465 {
		tlsconfig := &tls.Config{
			InsecureSkipVerify: true,
			ServerName:         host,
		}

		conn, err := tls.Dial("tcp", addr, tlsconfig)
		if err != nil {
			return fmt.Errorf("failed to connect SSL TLS: %w", err)
		}

		client, err := smtp.NewClient(conn, host)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
		defer client.Close()

		if auth != nil {
			if err = client.Auth(auth); err != nil {
				return fmt.Errorf("SMTP auth failed: %w", err)
			}
		}

		if err = client.Mail(from); err != nil {
			return err
		}
		if err = client.Rcpt(to); err != nil {
			return err
		}

		w, err := client.Data()
		if err != nil {
			return err
		}
		_, err = w.Write([]byte(message))
		if err != nil {
			return err
		}
		return w.Close()
	}

	return smtp.SendMail(addr, auth, from, []string{to}, []byte(message))
}