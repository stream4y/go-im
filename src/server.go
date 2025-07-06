package main

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

type Server struct {
	Ip   string
	Port int

	//在线用户列表
	OnlineMap map[string]*User
	mapLock   sync.RWMutex

	//消息广播channel
	Message chan string
}

// 创建Server对象
func NewServer(ip string, port int) *Server {
	server := &Server{
		Ip:        ip,
		Port:      port,
		OnlineMap: make(map[string]*User),
		Message:   make(chan string),
	}

	return server
}

// 广播消息
func (this *Server) BroadCast(user *User, msg string) {
	sendMsg := "[" + user.Name + "]" + user.Name + "：" + msg
	this.Message <- sendMsg
}

// 监听 Message 广播消息 channel 的 goroutine，一旦有消息就发送给所有在线用户
func (this *Server) ListenMsg() {
	for {
		msg := <-this.Message

		// 发给所有用户
		this.mapLock.Lock()
		for _, client := range this.OnlineMap {
			client.C <- msg
		}
		this.mapLock.Unlock()
	}
}

func (this *Server) Handler(conn net.Conn) {
	//fmt.Println("连接建立成功，IP：", conn.RemoteAddr())

	user := NewUser(conn, this)
	user.OnLine()

	// 监听用户是否活跃
	isLive := make(chan bool)

	// 接受客户端发送的消息 并进行广播
	go func() {
		buf := make([]byte, 4096)

		for {
			cnt, err := conn.Read(buf)
			if cnt == 0 {
				user.OffLine()
				return
			}
			if err != nil && err != io.EOF {
				fmt.Println("conn read err:", err)
				return
			}

			// 获取用户消息
			msg := string(buf[:cnt-1])

			user.DoMessage(msg)

			isLive <- true
		}
	}()

	// 当前handler阻塞
	for {
		select {
		// 是否活跃
		case <-isLive:
			// 用户活跃，不做处理

		// 已经超时 强制下线
		case <-time.After(time.Minute * 10):
			user.SendMsg("system: 已超时，强制下线")

			// 清理资源，断开连接
			close(user.C)
			err := conn.Close()
			if err != nil {
				fmt.Println("close conn err:", err)
				return
			}

			// 退出当前handler
			return
		}
	}
}

// 启动服务器的接口
func (this *Server) Run() {
	listen, err := net.Listen("tcp", fmt.Sprintf("%s:%d", this.Ip, this.Port))
	if err != nil {
		fmt.Println("net.Listen err：", err)
		return
	}

	defer listen.Close()

	// 启动goroutine监听Message
	go this.ListenMsg()

	for {
		conn, err := listen.Accept()
		if err != nil {
			fmt.Println("listen.Accept err：", err)
			continue
		}

		go this.Handler(conn)
	}
}
