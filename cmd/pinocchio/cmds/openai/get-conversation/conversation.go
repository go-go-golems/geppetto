package get_conversation

type NextData struct {
	Props struct {
		PageProps struct {
			SharedConversationId string `json:"sharedConversationId"`
			ServerResponse       struct {
				ServerResponseData `json:"data"`
			} `json:"serverResponse"`
			Model           map[string]interface{} `json:"model"`
			ModerationState map[string]interface{} `json:"moderation_state"`
		} `json:"pageProps"`
	} `json:"props"`
}

type ServerResponseData struct {
	Title              string         `json:"title"`
	CreateTime         float64        `json:"create_time"`
	UpdateTime         float64        `json:"update_time"`
	LinearConversation []Conversation `json:"linear_conversation"`
}

type Author struct {
	Role     string                 `json:"role"`
	Metadata map[string]interface{} `json:"metadata"`
}

type Content struct {
	ContentType string   `json:"content_type"`
	Parts       []string `json:"parts"`
}

type Message struct {
	ID        string                 `json:"id"`
	Author    Author                 `json:"author"`
	Content   Content                `json:"content"`
	Status    string                 `json:"status"`
	EndTurn   bool                   `json:"end_turn"`
	Weight    float64                `json:"weight"`
	Metadata  map[string]interface{} `json:"metadata"`
	Recipient string                 `json:"recipient"`
}

type Conversation struct {
	ID       string                 `json:"id"`
	Message  Message                `json:"message"`
	Parent   string                 `json:"parent"`
	Children []string               `json:"children"`
	Metadata map[string]interface{} `json:"metadata"`
}
