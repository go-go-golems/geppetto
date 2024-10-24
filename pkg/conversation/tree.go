package conversation

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"os"
)

type NodeID uuid.UUID

func (id NodeID) MarshalJSON() ([]byte, error) {
	return json.Marshal(uuid.UUID(id))
}

func (id *NodeID) UnmarshalJSON(data []byte) error {
	var uuid uuid.UUID
	if err := json.Unmarshal(data, &uuid); err != nil {
		return err
	}
	*id = NodeID(uuid)
	return nil
}

func (id NodeID) String() string {
	return uuid.UUID(id).String()
}

func NewNodeID() NodeID {
	return NodeID(uuid.New())
}

// Intermediate representation for unmarshaling.
type messageNodeAlias struct {
	ID          NodeID                 `json:"id"`
	ParentID    NodeID                 `json:"parentID"`
	Content     json.RawMessage        `json:"content"`
	Metadata    map[string]interface{} `json:"metadata"`
	ContentType ContentType            `json:"contentType"`
}

// UnmarshalJSON custom unmarshaler for Message.
func (mn *Message) UnmarshalJSON(data []byte) error {
	var mna messageNodeAlias
	if err := json.Unmarshal(data, &mna); err != nil {
		return err
	}

	// Determine the type of content based on ContentType.
	switch mna.ContentType {
	case ContentTypeChatMessage:
		var content *ChatMessageContent
		if err := json.Unmarshal(mna.Content, &content); err != nil {
			return err
		}
		mn.Content = content
	case ContentTypeToolUse:
		var content *ToolUseContent
		if err := json.Unmarshal(mna.Content, &content); err != nil {
			return err
		}
		mn.Content = content
	case ContentTypeToolResult:
		var content *ToolResultContent
		if err := json.Unmarshal(mna.Content, &content); err != nil {
			return err
		}
		mn.Content = content
	case ContentTypeImage:
		var content *ImageContent
		if err := json.Unmarshal(mna.Content, &content); err != nil {
			return err
		}
		mn.Content = content
	default:
		return errors.New("unknown content type")
	}

	mn.ID = mna.ID
	mn.ParentID = mna.ParentID
	mn.Metadata = mna.Metadata
	return nil
}

// ConversationTree represents a tree-like structure for storing and managing conversation messages.
//
// The tree consists of nodes (messages) connected by parent-child links. These relationships are done
// through the parent ID field in each message. The root node is the starting point of the conversation,
// and each node can have multiple children. The tree allows for traversing the conversation in various ways.
//
// Node relationships are stored in the Message datastructure as `Children []*Message`.
//
// Each node has a unique ID, and the tree keeps track of the root node ID and the last inserted node ID.
type ConversationTree struct {
	Nodes  map[NodeID]*Message
	RootID NodeID
	LastID NodeID
}

func NewConversationTree() *ConversationTree {
	return &ConversationTree{
		Nodes: make(map[NodeID]*Message),
	}
}

var NullNode NodeID = NodeID(uuid.Nil)

// InsertMessages adds new messages to the conversation tree.
// It updates the root ID if the tree is empty and sets the last inserted node ID.
// If a message has a parent ID that exists in the tree, it is added as a child of that parent node.
func (ct *ConversationTree) InsertMessages(msgs ...*Message) {
	for _, msg := range msgs {
		ct.Nodes[msg.ID] = msg
		if ct.RootID == NullNode {
			ct.RootID = msg.ID
		}
		ct.LastID = msg.ID

		if parent, exists := ct.Nodes[msg.ParentID]; exists {
			parent.Children = append(parent.Children, msg)
		}
	}
}

// AttachThread attaches a conversation thread to a specified parent message.
// It updates the parent IDs of the messages in the thread to link them to the parent message.
// The last message in the thread becomes the new last inserted node ID.
func (ct *ConversationTree) AttachThread(parentID NodeID, thread Conversation) {
	for _, msg := range thread {
		msg.ParentID = parentID
		ct.Nodes[msg.ID] = msg
		if ct.RootID == NullNode {
			ct.RootID = msg.ID
		}
		ct.LastID = msg.ID

		if parent, exists := ct.Nodes[msg.ParentID]; exists {
			parent.Children = append(parent.Children, msg)
		}
		parentID = msg.ID
	}
}

// AppendMessages appends a conversation thread to the end of the tree.
// It attaches the thread to the last inserted node in the tree, making it the parent of the thread.
// The messages in the thread are inserted as nodes, extending the parent-child chain.
func (ct *ConversationTree) AppendMessages(thread Conversation) {
	ct.AttachThread(ct.LastID, thread)
}

// PrependThread prepends a conversation thread to the beginning of the tree.
// It updates the root ID to the first message in the thread and adjusts the parent-child relationships accordingly.
// The previous root node becomes a child of the new root node.
func (ct *ConversationTree) PrependThread(thread Conversation) {
	prevRootID := ct.RootID
	newRootID := NullNode
	for _, msg := range thread {
		ct.Nodes[msg.ID] = msg
		ct.RootID = msg.ID
		newRootID = msg.ID
		// not setting LastID on purpose

		if parent, exists := ct.Nodes[msg.ParentID]; exists {
			parent.Children = append(parent.Children, msg)
		}
	}

	if prevRootID != NullNode {
		if prevRoot, exists := ct.Nodes[prevRootID]; exists {
			prevRoot.ParentID = newRootID
		}
	}
}

// FindSiblings returns the IDs of all sibling messages for a given message ID.
// Sibling messages are the nodes that share the same parent as the given message.
func (ct *ConversationTree) FindSiblings(id NodeID) []NodeID {
	node, exists := ct.Nodes[id]
	if !exists {
		return nil
	}

	parent, exists := ct.Nodes[node.ParentID]
	if !exists {
		return nil
	}

	var siblings []NodeID
	for _, sibling := range parent.Children {
		if sibling.ID != id {
			siblings = append(siblings, sibling.ID)
		}
	}

	return siblings
}

// FindChildren returns the IDs of all child messages for a given message ID.
func (ct *ConversationTree) FindChildren(id NodeID) []NodeID {
	node, exists := ct.Nodes[id]
	if !exists {
		return nil
	}

	var children []NodeID
	for _, child := range node.Children {
		children = append(children, child.ID)
	}

	return children
}

// GetConversationThread retrieves the linear conversation thread from root to the specified message.
func (ct *ConversationTree) GetConversationThread(id NodeID) Conversation {
	var thread Conversation
	for uuid.UUID(id) != uuid.Nil {
		node, exists := ct.Nodes[id]
		if !exists {
			break
		}
		thread = append([]*Message{node}, thread...)
		id = node.ParentID
	}
	return thread
}

// GetLeftMostThread returns the thread starting from a given message ID by always choosing the first child.
// It traverses the tree downwards, selecting the leftmost child at each level, until a leaf node is reached.
// The returned conversation is a linear sequence of messages from the given message to the leftmost leaf.
func (ct *ConversationTree) GetLeftMostThread(id NodeID) Conversation {
	var thread Conversation
	for id != NullNode {
		node, exists := ct.Nodes[id]
		if !exists {
			break
		}
		thread = append(thread, node)
		if len(node.Children) > 0 {
			id = node.Children[0].ID
		} else {
			id = NullNode
		}
	}
	return thread
}

func (ct *ConversationTree) SaveToFile(filename string) error {
	data, err := json.MarshalIndent(ct, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func (ct *ConversationTree) LoadFromFile(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, ct)
}

func (ct *ConversationTree) GetMessageByID(id NodeID) (*Message, bool) {
	ret, exists := ct.Nodes[id]
	return ret, exists
}
