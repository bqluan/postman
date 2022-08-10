package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/cheggaaa/pb/v3"
	"github.com/spf13/pflag"
	"gopkg.in/gomail.v2"
)

var (
	host      string
	port      int
	username  string
	password  string
	from      string
	subject   string
	tmplFile  string
	attach    string
	recipient string

	tmpl            *template.Template
	saveStateLock   sync.RWMutex
	saveStateCalled bool
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
	pflag.StringVarP(&host, "host", "h", "", "SMTP 服务器主机名或 IP 地址，例如 smtp.example.com（必填）")
	pflag.IntVar(&port, "port", 25, "SMTP 服务器端口号")
	pflag.StringVarP(&username, "username", "u", "", "用来登录 SMTP 服务器的用户名（必填）")
	pflag.StringVarP(&password, "password", "p", "", "用来登录 SMTP 服务器的密码（必填）")
	pflag.StringVarP(&tmplFile, "template", "t", "", "邮件正文模板文件名，请确保文件以 UTF-8 编码（必填）")
	pflag.StringVarP(&attach, "attach", "a", "", "附件文件名")
	pflag.StringVarP(&recipient, "recipient", "r", "", "收件人 CSV 文件名，请确保文件以 UTF-8 编码（必填）")
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
		fmt.Println("请填写 SMTP 服务器主机名或 IP 地址")
		pflag.Usage()
		os.Exit(1)
	}
	if username == "" {
		fmt.Println("请填写用来登录 SMTP 服务器的用户名")
		pflag.Usage()
		os.Exit(1)
	}
	if password == "" {
		fmt.Println("请填写用来登录 SMTP 服务器的密码")
		pflag.Usage()
		os.Exit(1)
	}
	if tmplFile == "" {
		fmt.Println("请填写邮件正文模板文件名")
		pflag.Usage()
		os.Exit(1)
	}
	if recipient == "" {
		fmt.Println("请填写收件人 CSV 文件名")
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

	d := gomail.NewDialer(host, port, username, password)
	return d.DialAndSend(m)
}

func saveState() error {
	saveStateLock.Lock()
	defer saveStateLock.Unlock()

	if saveStateCalled {
		return nil
	} else {
		saveStateCalled = true
	}

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

func loadRecipients() error {
	log.Printf("Loading recipients from %s", recipients)

	f, err := os.Open(recipient)
	if err != nil {
		return err
	}
	defer f.Close()

	r := csv.NewReader(f)
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		recipients = append(recipients, &Recipient{
			Name:   record[0],
			Title:  record[1],
			Email:  record[2],
			Status: record[3],
		})
	}

	log.Printf("%d recipients loaded", len(recipients))
	return nil
}

func run() error {
	stopCh := make(chan struct{})

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt)
	go func() {
		<-shutdownCh
		close(stopCh)
		<-shutdownCh
		saveState()
		os.Exit(1) // second signal. Exit directly.
	}()

	stoppedCh, err := nonBlockingRun(stopCh)
	if err != nil {
		return err
	}

	<-stoppedCh

	return saveState()
}

func nonBlockingRun(stopCh <-chan struct{}) (<-chan struct{}, error) {
	stoppedCh := make(chan struct{})

	go func() {
		bar := pb.StartNew(len(recipients))
		bar.SetWriter(os.Stdout)

	loop:
		for _, reci := range recipients {
			select {
			case <-stopCh:
				break loop
			default:
			}

			// skip recipient already sent
			if reci.Status == "ok" {
				bar.Increment()
				log.Printf("Skipping %s, already sent", reci.Email)
				continue loop
			}

			log.Printf("Sending email to %s...", reci.Email)
			if err := sendTo(reci); err != nil {
				log.Printf("Error sending email to %s: %v", reci.Email, err)
				continue loop
			}

			log.Printf("Email sent to %s", reci.Email)
			reci.Status = "ok"
			bar.Increment()
		}

		bar.Finish()
		close(stoppedCh)
	}()

	return stoppedCh, nil
}

func main() {
	pflag.Parse()
	makeSureRequiredFlagsExist()

	// load template
	log.Printf("Loading template from %s", tmplFile)
	buf, err := ioutil.ReadFile(tmplFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	tmpl, err = template.New("body").Parse(string(buf))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// load recipients
	if err = loadRecipients(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err = run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
