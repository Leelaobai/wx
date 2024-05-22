package main

import (
	"fmt"
	"strings"
	persistence "wx/Persistence"
	"wx/chat"

	"github.com/eatmoreapple/openwechat"
)

var self *openwechat.Self

func CallAIAndReply(msg *openwechat.Message) {
	userId := ""
	var sender *openwechat.User
	var err error
	var question string
	if msg.IsSendByGroup() && msg.IsAt() {
		sender, err = msg.SenderInGroup()
		if err != nil {
			fmt.Println("没有找到合适的消息发送者：", err.Error(), "\nmsg: ", msg.String())
			return
		}
		userId = sender.ID()

		atFlag := "@" + openwechat.FormatEmoji(self.NickName)
		if !strings.Contains(msg.Content, atFlag) {
			return
		}
		question = strings.ReplaceAll(msg.Content, atFlag, "")
	} else {
		// sender, err = msg.Sender()
		// if err != nil {
		// 	fmt.Println("没有找到合适的消息发送者：", err.Error(), "\nmsg: ", msg.String())
		// 	return
		// }
		// userId = sender.ID()
		return
	}

	answer, err := chat.ChatForAI(userId, question)
	if err != nil {
		fmt.Println("访问AI出错：", err.Error(), "\nmsg: ", msg.String())
		answer = "当前服务不可用，请稍后再试。"
	}
	atText := "@" + sender.NickName
	replyText := atText + "\u2005" + answer

	_, err = msg.ReplyText(replyText)
	if err != nil {
		fmt.Printf("回复消息出错: %v \n %s", err, "msg: "+msg.String())
	}
}

func main() {
	err := persistence.InitDB()
	if err != nil {
		fmt.Println("数据库初始化失败：", err.Error())
		return
	}
	reloadStorage := openwechat.NewFileHotReloadStorage("storage.json")

	chat.Init("sk-123456")

	defer reloadStorage.Close()

	bot := openwechat.DefaultBot(openwechat.Desktop) // 桌面模式

	// 注册消息处理函数
	bot.MessageHandler = func(msg *openwechat.Message) {
		if msg.IsText() {
			go CallAIAndReply(msg)
		}
	}
	// 注册登陆二维码回调
	bot.UUIDCallback = openwechat.PrintlnQrcodeUrl

	// 登陆
	if err = bot.HotLogin(reloadStorage, openwechat.NewRetryLoginOption()); err != nil {
		fmt.Println(err)
		return
	}

	// 获取登陆的用户
	self, err = bot.GetCurrentUser()
	if err != nil {
		fmt.Println(err)
		return
	}

	// 获取所有的好友
	friends, err := self.Friends()
	fmt.Println(friends, err)

	// 获取所有的群组
	groups, err := self.Groups()
	fmt.Println(groups, err)

	// 阻塞主goroutine, 直到发生异常或者用户主动退出
	bot.Block()
}
