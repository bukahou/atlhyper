package channel

import (
	"bufio"
	"context"
	"encoding/base64"
	"mime"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func testEmailConfig(host string, port int) EmailConfig {
	return EmailConfig{
		SMTPHost: host, SMTPPort: port, SMTPUser: "ops@example.com", SMTPPassword: "placeholder-not-a-real-password",
		FromAddress: "ops@example.com", ToAddresses: []string{"a@example.com", "b@example.com"},
	}
}

func headerValue(t *testing.T, message, name string) string {
	t.Helper()
	head, _, _ := strings.Cut(message, "\r\n\r\n")
	for _, line := range strings.Split(head, "\r\n") {
		if strings.HasPrefix(line, name+": ") {
			return strings.TrimPrefix(line, name+": ")
		}
	}
	t.Fatalf("没有 %s 头：\n%s", name, head)
	return ""
}

// 邮件主题按 RFC 2047 编码：头部只有 ASCII，解码后与原文一致
func TestBuildMailMessage_SubjectEncodedPerRFC2047(t *testing.T) {
	notifier := NewEmailNotifier(testEmailConfig("smtp.example.com", 587))
	subject := "[prod] 高 · shop/deployment/api · 风险持续偏高"
	message := notifier.buildMailMessage(subject, "第一行\n第二行", time.Date(2026, 11, 15, 10, 0, 0, 0, time.UTC))

	head, body, found := strings.Cut(message, "\r\n\r\n")
	if !found {
		t.Fatal("邮件应有头部与正文")
	}
	for _, char := range head {
		if char > 127 {
			t.Fatalf("头部出现了非 ASCII 字符：\n%s", head)
		}
	}
	encoded := headerValue(t, message, "Subject")
	if !strings.HasPrefix(encoded, "=?UTF-8?") {
		t.Errorf("主题应是 RFC 2047 编码字，实际 %q", encoded)
	}
	decoded, err := new(mime.WordDecoder).DecodeHeader(encoded)
	if err != nil || decoded != subject {
		t.Errorf("主题解码后应与原文一致，实际 %q (%v)", decoded, err)
	}

	if headerValue(t, message, "To") != "a@example.com, b@example.com" || headerValue(t, message, "From") != "ops@example.com" {
		t.Errorf("收发件人头不对：\n%s", head)
	}
	if headerValue(t, message, "Content-Transfer-Encoding") != "base64" {
		t.Errorf("正文应按 base64 传输：\n%s", head)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(body, "\r\n", ""))
	if err != nil || string(raw) != "第一行\r\n第二行" {
		t.Errorf("正文解码后不对：%q (%v)", raw, err)
	}
	for _, line := range strings.Split(body, "\r\n") {
		if len(line) > 76 {
			t.Errorf("正文行超过 76 个字符：%d", len(line))
		}
	}
}

// 主题里的换行不能变成新的邮件头
func TestBuildMailMessage_SubjectCannotInjectHeaders(t *testing.T) {
	notifier := NewEmailNotifier(testEmailConfig("smtp.example.com", 587))
	message := notifier.buildMailMessage("标题\r\nBcc: attacker@example.com", "正文", time.Now())
	head, _, _ := strings.Cut(message, "\r\n\r\n")
	for _, line := range strings.Split(head, "\r\n") {
		if strings.HasPrefix(strings.ToLower(line), "bcc:") {
			t.Fatalf("主题里的换行注入了新的头：\n%s", head)
		}
	}
}

func TestBuildMailMessage_FromFallsBackToSMTPUser(t *testing.T) {
	config := testEmailConfig("smtp.example.com", 587)
	config.FromAddress = ""
	message := NewEmailNotifier(config).buildMailMessage("s", "b", time.Now())
	if headerValue(t, message, "From") != "ops@example.com" {
		t.Errorf("没填发件人时应使用 SMTP 账号")
	}
}

// fakeSMTPServer 只在本机回环地址上监听的最小 SMTP 服务，记录收到的指令
type fakeSMTPServer struct {
	listener net.Listener
	mu       sync.Mutex
	commands []string
	data     string
	done     chan struct{}
}

func startFakeSMTPServer(t *testing.T) *fakeSMTPServer {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &fakeSMTPServer{listener: listener, done: make(chan struct{})}
	go server.serveOne()
	t.Cleanup(func() { listener.Close() })
	return server
}

func (s *fakeSMTPServer) port() int { return s.listener.Addr().(*net.TCPAddr).Port }

func (s *fakeSMTPServer) serveOne() {
	defer close(s.done)
	conn, err := s.listener.Accept()
	if err != nil {
		return
	}
	defer conn.Close()
	reader := bufio.NewReader(conn)
	reply := func(line string) { conn.Write([]byte(line + "\r\n")) }

	reply("220 fake ESMTP")
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		command := strings.TrimRight(line, "\r\n")
		s.mu.Lock()
		s.commands = append(s.commands, command)
		s.mu.Unlock()
		switch {
		case strings.HasPrefix(command, "EHLO"):
			reply("250-fake")
			reply("250 AUTH PLAIN")
		case strings.HasPrefix(command, "AUTH"):
			reply("235 ok")
		case command == "DATA":
			reply("354 go ahead")
			var body strings.Builder
			for {
				dataLine, err := reader.ReadString('\n')
				if err != nil || dataLine == ".\r\n" {
					break
				}
				body.WriteString(dataLine)
			}
			s.mu.Lock()
			s.data = body.String()
			s.mu.Unlock()
			reply("250 queued")
		case command == "QUIT":
			reply("221 bye")
			return
		default:
			reply("250 ok")
		}
	}
}

func TestEmailNotifier_SendsThroughSMTPDialogue(t *testing.T) {
	server := startFakeSMTPServer(t)
	notifier := NewEmailNotifier(testEmailConfig("127.0.0.1", server.port()))

	if err := notifier.Send(context.Background(), &Message{Subject: "告警标题", Body: "正文"}); err != nil {
		t.Fatalf("发送应成功: %v", err)
	}
	<-server.done

	server.mu.Lock()
	defer server.mu.Unlock()
	joined := strings.Join(server.commands, "\n")
	for _, want := range []string{"MAIL FROM:<ops@example.com>", "RCPT TO:<a@example.com>", "RCPT TO:<b@example.com>", "DATA", "QUIT"} {
		if !strings.Contains(joined, want) {
			t.Errorf("SMTP 对话里缺少 %q：\n%s", want, joined)
		}
	}
	if !strings.Contains(joined, "AUTH PLAIN") {
		t.Errorf("应当认证：\n%s", joined)
	}
	if !strings.Contains(server.data, "Subject: =?UTF-8?") {
		t.Errorf("邮件内容里应有编码后的主题：\n%s", server.data)
	}
}

// 对端接受连接后一言不发：整体超时到了必须返回，不能一直挂着
func TestEmailNotifier_TotalTimeout(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		if conn, err := listener.Accept(); err == nil {
			accepted <- conn // 握着连接不回话
		}
	}()
	defer func() {
		select {
		case conn := <-accepted:
			conn.Close()
		default:
		}
	}()

	notifier := NewEmailNotifier(testEmailConfig("127.0.0.1", listener.Addr().(*net.TCPAddr).Port))
	notifier.totalTimeout = 200 * time.Millisecond

	finished := make(chan error, 1)
	go func() { finished <- notifier.Send(context.Background(), &Message{Subject: "s", Body: "b"}) }()
	select {
	case err := <-finished:
		if err == nil {
			t.Error("对端不回话时应当超时失败")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("发送没有在整体超时内返回")
	}
}

func TestEmailNotifier_ErrorsDoNotContainPassword(t *testing.T) {
	// 端口上没有服务：连接失败
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	config := testEmailConfig("127.0.0.1", port)
	err := NewEmailNotifier(config).Send(context.Background(), &Message{Subject: "s", Body: "b"})
	if err == nil {
		t.Fatal("应当失败")
	}
	if strings.Contains(err.Error(), config.SMTPPassword) {
		t.Errorf("错误信息里出现了密码：%s", err)
	}
	if !strings.Contains(err.Error(), strconv.Itoa(port)) && !strings.Contains(err.Error(), "连接") {
		t.Errorf("错误信息应说明是连接失败：%s", err)
	}
}
