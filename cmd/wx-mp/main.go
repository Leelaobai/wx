package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"wx/chat"
	"wx/cmd/wx-mp/config"
	persistence "wx/persistence"

	wechat "github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/officialaccount"
	"github.com/silenceper/wechat/v2/officialaccount/message"

	offConfig "github.com/silenceper/wechat/v2/officialaccount/config"

	"github.com/silenceper/wechat/v2/cache"
)

var nonceMap sync.Map
var officialAccount *officialaccount.OfficialAccount

func serveWechat(rw http.ResponseWriter, req *http.Request) {
	//b, err := ioutil.ReadAll(req.Body)
	//if err != nil {
	//	log.Println("读取body失败：", err)
	//	return
	//}
	//log.Println("读取body：", string(b))
	nonce := req.URL.Query()["nonce"][0]
	_, loaded := nonceMap.LoadOrStore(nonce, true)
	if loaded {
		log.Println("收到已经存在的nonce：", req.URL.String())
		return
	}

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

	text := message.NewText("success")

	if msg.MsgType == "text" {

		go func() {
			greply, err := chat.ChatForAI(string(msg.FromUserName), msg.Content)
			if err != nil {
				log.Println("ERROR: ", err)
				greply = "请稍后重试。"
			}
			manager := officialAccount.GetCustomerMessageManager()
			err = manager.Send(message.NewCustomerTextMessage(string(msg.FromUserName), greply))
			if err != nil {
				log.Println("发送客服消息失败：", err.Error())
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
