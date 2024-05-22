package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
	"wx/chat"
	"wx/cmd/wx-mp/config"
	persistence "wx/persistence"

	wechat "github.com/silenceper/wechat/v2"
	"github.com/silenceper/wechat/v2/officialaccount/message"

	offConfig "github.com/silenceper/wechat/v2/officialaccount/config"

	"github.com/silenceper/wechat/v2/cache"
)

var nonceMap sync.Map

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
	officialAccount := wc.GetOfficialAccount(cfg)

	server := officialAccount.GetServer(req, rw)
	server.SetMessageHandler(handler)

	err := server.Serve()
	if err != nil {
		log.Println("处理消息失败: ", err)
		return
	}
	server.Send()

	/*
		go func() {
			//ta := time.After(1 * time.Millisecond)
			//select {
			//case ok := <-serverDone:
			//	if ok {
			//		err := server.Send()
			//		if err != nil {
			//			log.Println("发送消息失败：", err)
			//			return
			//		}
			//	}
			//	break
			//case <-ta:
			//<-serverDone
			//server.String("")
			//err := server.Send()
			//if err != nil {
			//	log.Println("发送success消息失败：", err)
			//	return
			//}
			//log.Println("发送success消息成功：")

			// 判定消息类型
			msgTypeTag := `<MsgType><![CDATA[`
			index := strings.Index(string(b), msgTypeTag) + len(msgTypeTag)

			// 调用客服消息
			accessToken, err := officialAccount.GetAccessToken()
			if err != nil {
				log.Println("获取AccessToken失败：", err)
				return
			}
			log.Println("获取AccessToken成功：", accessToken)


			jsonStr := ""
			url := "https://api.weixin.qq.com/cgi-bin/message/custom/send?access_token=" + accessToken

			if string(b)[index:index+4] != "text" {
				jsonStr = "ChatGPT Bot 仅支持文字内容"
			} else {

				contentFirstTag := `<Content><![CDATA[`
				contentLastTag := `]]></Content>`

				contentFirstIndex := strings.Index(string(b), contentFirstTag) + len(contentFirstTag)
				contentLastIndex := strings.Index(string(b), contentLastTag)

				reqMsg:=string(b)[contentFirstIndex:contentLastIndex]

				openid := server.Query("openid")
				log.Println("获得openid成功：", openid)

				//reqMsg := string(server.RequestRawXMLMsg)

				content, err := gtp.Completions(reqMsg)
				if err != nil {
					log.Println("访问chatGPT失败：", err)
					return
				}
				log.Println("访问chatGPT成功：", content)

				jsonStr = `{
				"touser":"OPENID",
				"msgtype":"text",
				"text":
				{
					"content":"CONTENT"
				}
			}`

				jsonStr = strings.Replace(jsonStr, "OPENID", openid, 1)
				jsonStr = strings.Replace(jsonStr, "CONTENT", content, 1)
			}
			res, err := http.Post(url, "application/json", bytes.NewReader([]byte(jsonStr)))
			if err != nil {
				log.Println("发送客服消息失败：", err)
				return
			}
			bbb,_:=ioutil.ReadAll(res.Body)
			log.Println("发送客服消息返回：", string(bbb))
			//	break
			//}
		}()

	*/
}

func handler(msg *message.MixMessage) *message.Reply {

	reply := "ChatGPT Bot 仅支持文字内容"
	if msg.MsgType == "text" {

		serverDone := make(chan bool, 1)
		greply := ""
		go func(c chan bool) {

			var err error
			greply, err = chat.ChatForAI(string(msg.FromUserName), msg.Content)
			if err != nil {
				log.Println("ERROR: ", err)
				c <- false
			}
			c <- true
		}(serverDone)

		ta := time.After(4800 * time.Millisecond)

		select {
		case <-serverDone:
			reply = strings.TrimSpace(greply)
			reply = strings.Trim(reply, "\n")
			break
		case <-ta:
			reply = "请20s后再试。"
			break
		}

	}
	text := message.NewText(reply)
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
