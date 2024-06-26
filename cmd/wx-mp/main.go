package main

import (
	"fmt"
	"log"
	"net/http"
	"wx/chat"
	"wx/cmd/wx-mp/config"
	persistence "wx/persistence"

	wechat "github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/officialaccount"
	"github.com/silenceper/wechat/v2/officialaccount/message"

	offConfig "github.com/silenceper/wechat/v2/officialaccount/config"

	"github.com/silenceper/wechat/v2/cache"
)

var officialAccount *officialaccount.OfficialAccount

func serveWechat(rw http.ResponseWriter, req *http.Request) {

	wc := wechat.NewWechat()
	memory := cache.NewMemory()
	cfg := &offConfig.Config{
		AppID:          config.LoadConfig().WxAppId,
		AppSecret:      config.LoadConfig().AppSecret,
		Token:          config.LoadConfig().WxToken,
		EncodingAESKey: config.LoadConfig().EncodingAESKey,
		Cache:          memory,
	}
	officialAccount = wc.GetOfficialAccount(cfg)

	server := officialAccount.GetServer(req, rw)
	server.SetMessageHandler(handler)

	err := server.Serve()
	if err != nil {
		log.Println("处理消息失败: ", err)
		return
	}
	server.Send()

}

func handler(msg *message.MixMessage) *message.Reply {

	text := message.NewText("")

	if msg.MsgType == "text" {

		go func() {
			greply, err := chat.ChatForAI(string(msg.FromUserName), msg.Content)
			if err != nil {
				log.Println("ERROR: ", err)
				greply = "请稍后重试。"
			}
			log.Println("返回的答复长为：", len(greply), "内容为：", greply)

			gr := []rune(greply)
			maxLen := 500
			i := 0
			grL := len(gr)
			for l := grL; l > 0; l = l - maxLen {
				rightIndex := i + maxLen
				if rightIndex > grL {
					rightIndex = grL
				}
				manager := officialAccount.GetCustomerMessageManager()
				err = manager.Send(message.NewCustomerTextMessage(string(msg.FromUserName), string(gr[i:rightIndex])))
				if err != nil {
					log.Println("发送客服消息失败：", err.Error())
					return
				}
				i += maxLen
			}

		}()
	}

	return &message.Reply{MsgType: message.MsgTypeText, MsgData: text}
}

func main() {
	chat.Init(config.LoadConfig().ApiKey)
	persistence.InitDB()
	port := ":" + config.LoadConfig().HttpPort
	http.HandleFunc("/", serveWechat)
	fmt.Println("wechat server listener at", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		fmt.Printf("start server error , err=%v", err)
	}
}
