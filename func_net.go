package antnet

import (
	"bytes"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strings"
)

func Send(msg *Message, fun func(msgque IMsgQue) bool) {
	if msg == nil {
		return
	}
	c := make(chan struct{})
	gmsgMapSync.Lock()
	gmsg := gmsgArray[gmsgId]
	gmsgArray[gmsgId+1] = &gMsg{c: c}
	gmsgId++
	gmsgMapSync.Unlock()
	gmsg.msg = msg
	gmsg.fun = fun
	close(gmsg.c)
}

func SendGroup(group string, msg *Message) {
	Send(msg, func(msgque IMsgQue) bool {
		return msgque.IsInGroup(group)
	})
}

func HttpGetWithBasicAuth(url, name, passwd string) (string, error, *http.Response) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", ErrHttpRequest, nil
	}
	req.SetBasicAuth(name, passwd)
	resp, err := client.Do(req)
	if err != nil {
		return "", ErrHttpRequest, nil
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", ErrHttpRequest, nil
	}
	resp.Body.Close()
	return string(body), nil, resp
}

func HttpGet(url string) (string, error, *http.Response) {
	resp, err := http.Get(url)
	if err != nil {
		return "", ErrHttpRequest, nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", ErrHttpRequest, resp
	}
	resp.Body.Close()
	return string(body), nil, resp
}

func HttpPost(url, form string) (string, error, *http.Response) {
	resp, err := http.Post(url, "application/x-www-form-urlencoded", strings.NewReader(form))
	if err != nil {
		return "", ErrHttpRequest, nil
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", ErrHttpRequest, resp
	}
	resp.Body.Close()
	return string(body), nil, resp
}

func HttpUpload(url, field, file string) (*http.Response, error) {
	buf := new(bytes.Buffer)
	writer := multipart.NewWriter(buf)
	formFile, err := writer.CreateFormFile(field, file)
	if err != nil {
		LogError("create form file failed:%s\n", err)
		return nil, err
	}

	srcFile, err := os.Open(file)
	if err != nil {
		LogError("%open source file failed:%s\n", err)
		return nil, err
	}
	defer srcFile.Close()
	_, err = io.Copy(formFile, srcFile)
	if err != nil {
		LogError("write to form file falied:%s\n", err)
		return nil, err
	}

	contentType := writer.FormDataContentType()
	writer.Close()
	resp, err := http.Post(url, contentType, buf)
	if err != nil {
		LogError("post failed:%s\n", err)
	}
	resp.Body.Close()
	return resp, err
}

func SendMail(user, password, host, to, subject, body, mailtype string) error {
	hp := strings.Split(host, ":")
	auth := smtp.PlainAuth("", user, password, hp[0])
	var content_type string
	if mailtype == "html" {
		content_type = "Content-Type: text/" + mailtype + "; charset=UTF-8"
	} else {
		content_type = "Content-Type: text/plain" + "; charset=UTF-8"
	}

	msg := []byte("To: " + to + "\r\nFrom: " + user + ">\r\nSubject: " + "\r\n" + content_type + "\r\n\r\n" + body)
	send_to := strings.Split(to, ";")
	err := smtp.SendMail(host, auth, user, send_to, msg)
	return err
}

var allIp []string

func GetSelfIp(ifnames ...string) []string {
	if allIp != nil {
		return allIp
	}
	inters, _ := net.Interfaces()
	if len(ifnames) == 0 {
		ifnames = []string{"eth", "lo", "eno", "无线网络连接", "本地连接", "以太网"}
	}

	filterFunc := func(name string) bool {
		for _, v := range ifnames {
			if strings.Index(name, v) != -1 {
				return true
			}
		}
		return false
	}

	for _, inter := range inters {
		if !filterFunc(inter.Name) {
			continue
		}
		addrs, _ := inter.Addrs()
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok {
				if ipnet.IP.To4() != nil {
					allIp = append(allIp, ipnet.IP.String())
				}
			}
		}
	}
	return allIp
}

func IsIntraIp(ip string) bool {
	if ip == "127.0.0.1" {
		return true
	}
	ips := strings.Split(ip, ".")
	ipA := ips[0]
	if ipA == "10" {
		return true
	}
	ipB := ips[1]

	if ipA == "192" {
		if ipB == "168" {
			return true
		}
	}

	if ipA == "172" {
		ipb := Atoi(ipB)
		if ipb >= 16 && ipb <= 31 {
			return true
		}
	}

	return false
}

func GetSelfIntraIp(ifnames ...string) (ips []string) {
	all := GetSelfIp(ifnames...)
	for _, v := range all {
		if IsIntraIp(v) {
			ips = append(ips, v)
		}
	}

	return
}

func GetSelfExtraIp(ifnames ...string) (ips []string) {
	all := GetSelfIp(ifnames...)
	for _, v := range all {
		if IsIntraIp(v) {
			continue
		}
		ips = append(ips, v)
	}

	return
}

func IPCanUse(ip string) bool {
	var err error
	for port := 1024; port < 65535; port++ {
		addr := Sprintf("%v:%v", ip, port)
		listen, err := net.Listen("tcp", addr)
		if err == nil {
			listen.Close()
			break
		} else if StrFind(err.Error(), "address is not valid") != -1 {
			return false
		}
	}
	return err == nil
}
