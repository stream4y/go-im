package main

import (
	"fmt"
	"net"
	"strings"
)

type User struct {
	Name   string
	Addr   string
	C      chan string
	conn   net.Conn
	server *Server // 当前用户属于哪个server
}

func NewUser(conn net.Conn, server *Server) *User {
	addr := conn.RemoteAddr().String()

	user := &User{
		Name:   addr,
		Addr:   addr,
		C:      make(chan string),
		conn:   conn,
		server: server,
	}

	//监听当前 user channel 消息的 goroutine
	go user.ListenMsg()

	return user
}

func (this *User) ListenMsg() {
	for {
		msg := <-this.C
		write, err := this.conn.Write([]byte(msg + "\n"))
		if err != nil {
			fmt.Println("msg send err：", err)
			return
		}
		if write != len(msg) {
			continue
		}
	}
}

// 用户上线
func (this *User) OnLine() {
	server := this.server
	// 加入onlineMap
	server.mapLock.Lock()
	server.OnlineMap[this.Name] = this
	server.mapLock.Unlock()

	// 广播当前用户已上线
	server.BroadCast(this, "已上线")
}

// 用户下线
func (this *User) OffLine() {
	server := this.server
	server.mapLock.Lock()
	delete(server.OnlineMap, this.Name)
	server.mapLock.Unlock()
	// 广播当前用户已上线
	server.BroadCast(this, "已下线")
}

// 用户处理消息
func (this *User) DoMessage(msg string) {
	if msg == "users" {
		this.Users()
	} else if strings.HasPrefix(msg, "rename") {
		// 格式  rename->张三
		this.Rename(strings.Split(msg, "->")[1])
	} else if strings.HasPrefix(msg, "send") {
		// 格式  send->张三->消息内容
		split := strings.Split(msg, "->")
		sendTo := split[1]
		sendMsg := split[2]

		user, ok := this.server.OnlineMap[sendTo]
		if !ok {
			this.SendMsg("system: 用户名[ " + sendTo + " ]不存在！请重新输入")
			return
		}

		if len(sendMsg) == 0 {
			this.SendMsg("system: 发送的消息内容为空！请重新输入")
			return
		}
		user.SendMsg(this.Name + ": " + sendMsg)
	} else {
		this.server.BroadCast(this, msg)
	}
}

// 给当前用户对应的客户端发送消息
func (this *User) SendMsg(msg string) {
	_, err := this.conn.Write([]byte(msg + "\n"))
	if err != nil {
		fmt.Println("send msg error:", err)
	}
}

// 查询用户所在server的用户列表
func (this *User) Users() {
	server := this.server
	server.mapLock.Lock()
	this.SendMsg("system: \n")
	for name, user := range server.OnlineMap {
		msg := "[" + user.Addr + "]" + name + ":" + "在线....\n"
		this.SendMsg(msg)
	}
	defer server.mapLock.Unlock()
}

// 修改用戶的用户名
func (this *User) Rename(name string) {
	this.server.mapLock.Lock()
	defer this.server.mapLock.Unlock()

	if _, exists := this.server.OnlineMap[name]; exists {
		this.SendMsg("system: 用户名" + "[ " + name + " ]" + "+已被占用")
		return
	}

	delete(this.server.OnlineMap, this.Name)
	this.Name = name
	this.server.OnlineMap[name] = this

	this.SendMsg("system: 用户名已更新为：" + name)
}
