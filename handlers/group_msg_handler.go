package handlers

import (
	"github.com/869413421/wechatbot/config"
	"github.com/869413421/wechatbot/gtp"
	"github.com/eatmoreapple/openwechat"
	"log"
	"os"
	"strings"
)

var _ MessageHandlerInterface = (*GroupMessageHandler)(nil)
var warnGroupFlag = true

// GroupMessageHandler 群消息处理
type GroupMessageHandler struct {
}

// handle 处理消息
func (g *GroupMessageHandler) handle(msg *openwechat.Message) error {
	// 自己加入群聊
	if selfJoinGroup(msg) {
		img, err := os.Open("vqilai.jpg")
		if err != nil {
			msg.ReplyText("家人们朋友们，我是【V起来】微信群聊版 ChatGPT机器人。很高兴和大家见面！你可以@我，提任何问题。\n\n防止失联，请一定收藏关注我的公众号哦[红包]\n https://mp.weixin.qq.com/s/n-zjrRsa8lNrzhZV9iFMww")
			return nil
		}
		msg.ReplyText("家人们朋友们，我是【V起来】微信群聊版 ChatGPT机器人。很高兴和大家见面！你可以@我，提任何问题。\n\n防止失联，请一定收藏关注我的公众号哦[红包]")
		msg.ReplyImage(img)
		return nil
	}
	// 别人加入群聊
	if joinGroup(msg) && config.Config.JoinGroupTip != "" {
		msg.ReplyText(config.Config.JoinGroupTip)
		return nil
	}
	if msg.IsText() {
		go g.ReplyText(msg)
		return nil
	}
	return nil
}

// NewGroupMessageHandler 创建群消息处理器
func NewGroupMessageHandler() MessageHandlerInterface {
	return &GroupMessageHandler{}
}

// ReplyText 发送文本消息到群
func (g *GroupMessageHandler) ReplyText(msg *openwechat.Message) error {
	// 不是@的不处理
	if !msg.IsAt() {
		return nil
	}

	// 接收群消息
	sender, err := msg.Sender()
	group := openwechat.Group{sender}

	log.Printf("Received Group %v Text Msg : %v", group.NickName, msg.Content)

	// 替换掉@文本，设置会话上下文，然后向GPT发起请求。
	replaceText := "@" + sender.Self.NickName
	requestText := strings.TrimSpace(strings.ReplaceAll(msg.Content, replaceText, ""))
	if requestText == "" {
		return nil
	}

	// 获取@我的用户
    groupSender, err := msg.SenderInGroup()
    if err != nil {
        log.Printf("get sender in group error :%v \n", err)
        return err
    }

    requestText = UserService.GetUserSessionContext(groupSender.NickName) + requestText
	reply, err := gtp.Completions(requestText)

	// 回复@我的用户
    reply = strings.TrimSpace(reply)
    reply = strings.Trim(reply, "\n")
    // 设置上下文
    UserService.SetUserSessionContext(groupSender.NickName, requestText, reply)
    atText := "@" + groupSender.NickName + " "

	if err != nil {
		log.Printf("gtp request error: %v \n", err)
		errorTip := atText + "机器人去美国找OpenAI超时了，我要回答下个问题了。"
		if reply == "400" {
			UserService.ClearUserSessionContext()
			log.Print("发生超时，上下文会话缓存清理")
		}
		if reply == "429" {
		    warnGroup(msg)
		    errorTip = errorTip + "!!!!!!!"
		}
		_, err = msg.ReplyText(errorTip)
		if err != nil {
			log.Printf("response group error: %v \n", err)
		}
		return err
	}
	if reply == "" {
		return nil
	}

	// 设置上下文
	UserService.SetUserSessionContext(groupSender.NickName, requestText, reply)

	if strings.Contains(reply, "\n") {
	    atText = atText + "\n"
	}
	replyText := atText + reply
	_, err = msg.ReplyText(replyText)
	if err != nil {
		log.Printf("response group error: %v \n", err)
	}
	return err
}

func warnGroup(msg *openwechat.Message) error{
	self, err := msg.Bot.GetCurrentUser()
	groups, err := self.Groups()
	topGroup := groups.GetByNickName(config.Config.AlarmGroupName)
	if topGroup != nil && warnGroupFlag{
		topGroup.SendText("keys已过期，尽快重置")
		warnUserFlg = false
	}
	return err
}

func joinGroup(m *openwechat.Message) bool {
	return m.IsSystem() &&(strings.Contains(m.Content, "加入了群聊") || strings.Contains(m.Content, "加入群聊")) && m.IsSendByGroup()
}

func selfJoinGroup(m *openwechat.Message) bool {
	return m.IsSystem() && (strings.Contains(m.Content, "你通过扫描二维码加入群聊") || strings.Contains(m.Content, "邀请你加入了群聊") || strings.Contains(m.Content, "邀请你"))
}
