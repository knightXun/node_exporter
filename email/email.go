package email

import (
	"gopkg.in/gomail.v2"
	"log"
	"fmt"
	"crypto/tls"
)

type EmailConfig struct {
	To []string
	Cc []string
	Header string
	Body string
	Subject string
	EmailServer string
	EmailUserName string
	EmailPassword string
}
func SendEmail(cfg EmailConfig) {
	m := gomail.NewMessage()

	m.SetAddressHeader("From", cfg.EmailUserName , "发件人")

	tos := []string{}
	for _, to := range cfg.To {
		tos = append(tos, m.FormatAddress(to, "收件人"))
	}
	m.SetHeader("To", tos...) // 收件人

	m.SetHeader("Subject", cfg.Subject)     // 主题
	m.SetBody("text/html", cfg.Body ) // 正文

	fmt.Println(cfg)

	d := gomail.NewDialer(cfg.EmailServer, 587, cfg.EmailUserName, cfg.EmailPassword)
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	if err := d.DialAndSend(m); err != nil {
		log.Println("发送失败", err)
		return
	}
}

