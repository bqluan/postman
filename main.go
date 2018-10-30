package main

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"

	"github.com/spf13/pflag"
	"gopkg.in/gomail.v2"
)

var (
	from     string
	subject  string
	host     string
	port     int
	username string
	password string
	tmplFile string

	tmpl *template.Template
)

type Recipient struct {
	Name  string
	Title string
	Email string
}

func init() {
	pflag.StringVarP(&from, "from", "f", "", "发信人邮件地址，例如 sender@example.com（必填）")
	pflag.StringVarP(&subject, "subject", "s", "", "邮件标题（必填）")
	pflag.StringVarP(&host, "host", "h", "", "SMTP服务器主机名或IP地址，例如 smtp.example.com（必填）")
	pflag.IntVar(&port, "port", 25, "SMTP服务器端口号")
	pflag.StringVarP(&username, "username", "u", "", "用来登录SMTP服务器的用户名（必填）")
	pflag.StringVarP(&password, "password", "p", "", "用来登录SMTP服务器的密码（必填）")
	pflag.StringVarP(&tmplFile, "template", "t", "", "邮件正文模板文件名，请确保文件以UTF-8编码（必填）")
}

func makeSureRequiredFlagsExist() {
	if from == "" {
		fmt.Println("请填写发信人邮件地址")
		pflag.Usage()
		os.Exit(1)
	}
	if subject == "" {
		fmt.Println("请填写邮件标题")
		pflag.Usage()
		os.Exit(1)
	}
	if host == "" {
		fmt.Println("请填写SMTP服务器主机名或IP地址")
		pflag.Usage()
		os.Exit(1)
	}
	if username == "" {
		fmt.Println("请填写用来登录SMTP服务器的用户名")
		pflag.Usage()
		os.Exit(1)
	}
	if password == "" {
		fmt.Println("请填写用来登录SMTP服务器的密码")
		pflag.Usage()
		os.Exit(1)
	}
	if tmplFile == "" {
		fmt.Println("请填写邮件正文模板文件名")
		pflag.Usage()
		os.Exit(1)
	}
}

func sendTo(r *Recipient) {
	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, r); err != nil {
		log.Fatal(err)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", r.Email)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", buf.String())

	d := gomail.NewPlainDialer(host, port, username, password)

	if err := d.DialAndSend(m); err != nil {
		log.Fatal(err)
	}
}

func main() {
	pflag.Parse()
	makeSureRequiredFlagsExist()

	buf, err := ioutil.ReadFile(tmplFile)
	if err != nil {
		log.Fatal(err)
	}
	tmpl, err = template.New("body").Parse(string(buf))
	if err != nil {
		log.Fatal(err)
	}

	r := &Recipient{
		Name:  "张三",
		Title: "处长",
		Email: "bingqian.luan@icloud.com",
	}

	sendTo(r)
}
