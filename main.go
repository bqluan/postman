package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"

	"github.com/spf13/pflag"
	"gopkg.in/gomail.v2"
)

var (
	from      string
	subject   string
	host      string
	port      int
	username  string
	password  string
	tmplFile  string
	attach    string
	recipient string

	tmpl *template.Template
)

type Recipient struct {
	Name   string
	Title  string
	Email  string
	Status string
}

var recipients []*Recipient

func init() {
	pflag.StringVarP(&from, "from", "f", "", "发信人邮件地址，例如 sender@example.com（必填）")
	pflag.StringVarP(&subject, "subject", "s", "", "邮件标题（必填）")
	pflag.StringVarP(&host, "host", "h", "", "SMTP服务器主机名或IP地址，例如 smtp.example.com（必填）")
	pflag.IntVar(&port, "port", 25, "SMTP服务器端口号")
	pflag.StringVarP(&username, "username", "u", "", "用来登录SMTP服务器的用户名（必填）")
	pflag.StringVarP(&password, "password", "p", "", "用来登录SMTP服务器的密码（必填）")
	pflag.StringVarP(&tmplFile, "template", "t", "", "邮件正文模板文件名，请确保文件以UTF-8编码（必填）")
	pflag.StringVarP(&attach, "attach", "a", "", "附件文件名")
	pflag.StringVarP(&recipient, "recipient", "r", "", "收件人CSV文件名，请确保文件以UTF-8编码（必填）")
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
	if recipient == "" {
		fmt.Println("请填写收件人CSV文件名")
		pflag.Usage()
		os.Exit(1)
	}
}

func sendTo(r *Recipient) error {
	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, r); err != nil {
		return err
	}

	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", r.Email)
	m.SetHeader("Subject", subject)
	m.SetBody("text/html", buf.String())
	if attach != "" {
		m.Attach(attach)
	}

	d := gomail.NewPlainDialer(host, port, username, password)

	return d.DialAndSend(m)
}

func saveState() error {
	var records [][]string
	for _, reci := range recipients {
		records = append(records, []string{reci.Name, reci.Title, reci.Email, reci.Status})
	}
	f, err := os.OpenFile(recipient, os.O_WRONLY, 600)
	if err != nil {
		return err
	}
	defer f.Close()
	return csv.NewWriter(f).WriteAll(records)
}

func main() {
	pflag.Parse()
	makeSureRequiredFlagsExist()

	// load template
	buf, err := ioutil.ReadFile(tmplFile)
	if err != nil {
		log.Fatal(err)
	}
	tmpl, err = template.New("body").Parse(string(buf))
	if err != nil {
		log.Fatal(err)
	}

	// load recipients
	f, err := os.Open(recipient)
	if err != nil {
		log.Fatal(err)
	}
	r := csv.NewReader(f)
	records, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	f.Close()
	for _, rec := range records {
		recipients = append(recipients, &Recipient{
			Name:   rec[0],
			Title:  rec[1],
			Email:  rec[2],
			Status: rec[3],
		})
	}

	fmt.Printf("Sending to %v recipients ", len(recipients))
	for _, reci := range recipients {
		if reci.Status == "ok" {
			fmt.Print("✓")
			continue
		}
		err := sendTo(reci)
		if err == nil {
			reci.Status = "ok"
			fmt.Print("✓")
		} else {
			fmt.Print("✗")
		}
		saveState()
	}
	fmt.Println()
}
