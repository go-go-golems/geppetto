package conversation

import (
	"encoding/json"
	"os"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

// Debugging counters for tree operations
var (
	attachThreadCallCounter = int64(0)
	appendMsgCallCounter    = int64(0)
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

func (id NodeID) MarshalYAML() (interface{}, error) {
	return uuid.UUID(id), nil
}

func (id *NodeID) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var uuid uuid.UUID
	if err := unmarshal(&uuid); err != nil {
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
	case ContentTypeError:
		var content *ErrorContent
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
// Returns an error if duplicate messages are detected or if self-reference cycles would be created.
func (ct *ConversationTree) InsertMessages(msgs ...*Message) error {
	for _, msg := range msgs {
		// Check for self-reference cycle
		if msg.ID == msg.ParentID {
			return errors.Errorf("self-reference cycle detected: message %s cannot be its own parent", msg.ID.String())
		}

		// Check for duplicate message with same content
		if existingMsg, exists := ct.Nodes[msg.ID]; exists {
			if existingMsg.Content.String() == msg.Content.String() {
				return errors.Errorf("duplicate message detected: message %s already exists with identical content", msg.ID.String())
			}
		}

		ct.Nodes[msg.ID] = msg
		if ct.RootID == NullNode {
			ct.RootID = msg.ID
		}
		ct.LastID = msg.ID

		if parent, exists := ct.Nodes[msg.ParentID]; exists {
			// Check if message is already a child to prevent duplicates
			alreadyChild := false
			for _, child := range parent.Children {
				if child.ID == msg.ID {
					alreadyChild = true
					break
				}
			}
			if !alreadyChild {
				parent.Children = append(parent.Children, msg)
			}
		}
	}
	return nil
}

// AttachThread attaches a conversation thread to a specified parent message.
// It updates the parent IDs of the messages in the thread to link them to the parent message.
// The last message in the thread becomes the new last inserted node ID.
// Returns an error if duplicate messages are detected or if self-reference cycles would be created.
func (ct *ConversationTree) AttachThread(parentID NodeID, thread Conversation) error {
	attachCallID := atomic.AddInt64(&attachThreadCallCounter, 1)
	attachStart := time.Now()

	log.Trace().
		Int64("attach_call_id", attachCallID).
		Str("parent_id", parentID.String()).
		Int("thread_length", len(thread)).
		Int("tree_node_count", len(ct.Nodes)).
		Str("tree_root_id", ct.RootID.String()).
		Str("tree_last_id", ct.LastID.String()).
		Msg("TREE ATTACH THREAD ENTRY - CRITICAL RECURSION POINT")

	originalNodeCount := len(ct.Nodes)
	processedMessages := 0

	for i, msg := range thread {
		msgStart := time.Now()
		oldParentID := msg.ParentID
		existingMsg, msgExists := ct.Nodes[msg.ID]
		parentNode, parentExists := ct.Nodes[msg.ParentID]

		log.Trace().
			Int64("attach_call_id", attachCallID).
			Int("message_index", i).
			Str("message_id", msg.ID.String()).
			Str("old_parent_id", oldParentID.String()).
			Str("new_parent_id", parentID.String()).
			Bool("message_exists", msgExists).
			Bool("parent_exists", parentExists).
			Int("existing_children_count", func() int {
				if parentNode != nil {
					return len(parentNode.Children)
				}
				return 0
			}()).
			Msg("PROCESSING MESSAGE IN ATTACH THREAD")

		if msgExists {
			log.Trace().
				Int64("attach_call_id", attachCallID).
				Str("message_id", msg.ID.String()).
				Str("existing_parent", existingMsg.ParentID.String()).
				Str("new_parent", parentID.String()).
				Bool("parent_will_change", existingMsg.ParentID != parentID).
				Int("existing_children", len(existingMsg.Children)).
				Str("existing_content", func() string {
					content := existingMsg.Content.String()
					if len(content) > 30 {
						return content[:30]
					}
					return content
				}()).
				Str("new_content", func() string {
					content := msg.Content.String()
					if len(content) > 30 {
						return content[:30]
					}
					return content
				}()).
				Msg("MESSAGE ALREADY EXISTS - POTENTIAL DUPLICATE/OVERWRITE")

			// If content is identical and parent isn't changing, return error for duplicate
			if existingMsg.Content.String() == msg.Content.String() && existingMsg.ParentID == parentID {
				log.Trace().
					Int64("attach_call_id", attachCallID).
					Str("message_id", msg.ID.String()).
					Msg("IDENTICAL MESSAGE DETECTED - RETURNING DUPLICATE ERROR")
				return errors.Errorf("duplicate message detected: message %s already exists with identical content and parent", msg.ID.String())
			}
		}

		// CRITICAL: This is where we modify the message parent ID
		// Prevent self-referencing cycles
		if msg.ID == parentID {
			log.Trace().
				Int64("attach_call_id", attachCallID).
				Str("message_id", msg.ID.String()).
				Str("parent_id", parentID.String()).
				Msg("PREVENTING SELF-REFERENCE CYCLE - RETURNING ERROR")
			return errors.Errorf("self-reference cycle detected: message %s cannot be attached to itself as parent", msg.ID.String())
		}

		msg.ParentID = parentID

		// CRITICAL: This overwrites existing messages with same ID
		ct.Nodes[msg.ID] = msg

		if ct.RootID == NullNode {
			log.Trace().
				Int64("attach_call_id", attachCallID).
				Str("message_id", msg.ID.String()).
				Msg("Setting new root ID")
			ct.RootID = msg.ID
		}

		// CRITICAL: Always update LastID
		oldLastID := ct.LastID
		ct.LastID = msg.ID

		log.Trace().
			Int64("attach_call_id", attachCallID).
			Str("old_last_id", oldLastID.String()).
			Str("new_last_id", ct.LastID.String()).
			Msg("Updated LastID")

		// CRITICAL: This is where we add to Children - potential for duplication
		if parent, exists := ct.Nodes[msg.ParentID]; exists {
			oldChildCount := len(parent.Children)

			// Check if this message is already in children to detect duplication
			alreadyChild := false
			for _, child := range parent.Children {
				if child.ID == msg.ID {
					alreadyChild = true
					break
				}
			}

			log.Trace().
				Int64("attach_call_id", attachCallID).
				Str("parent_id", msg.ParentID.String()).
				Str("child_id", msg.ID.String()).
				Int("old_child_count", oldChildCount).
				Bool("already_child", alreadyChild).
				Msg("ADDING TO PARENT CHILDREN - POTENTIAL DUPLICATE ISSUE")

			if alreadyChild {
				log.Trace().
					Int64("attach_call_id", attachCallID).
					Str("parent_id", msg.ParentID.String()).
					Str("child_id", msg.ID.String()).
					Msg("MESSAGE ALREADY IN PARENT CHILDREN - SKIPPING DUPLICATE")
			} else {
				parent.Children = append(parent.Children, msg)
				log.Trace().
					Int64("attach_call_id", attachCallID).
					Str("parent_id", msg.ParentID.String()).
					Str("child_id", msg.ID.String()).
					Msg("MESSAGE ADDED TO PARENT CHILDREN")
			}

			log.Trace().
				Int64("attach_call_id", attachCallID).
				Str("parent_id", msg.ParentID.String()).
				Int("new_child_count", len(parent.Children)).
				Int("child_count_increase", len(parent.Children)-oldChildCount).
				Msg("Parent children updated")
		} else {
			log.Trace().
				Int64("attach_call_id", attachCallID).
				Str("parent_id", msg.ParentID.String()).
				Str("message_id", msg.ID.String()).
				Msg("PARENT NOT FOUND - ORPHANED MESSAGE")
		}

		// CRITICAL: This sets up the chain for the next iteration
		parentID = msg.ID
		processedMessages++

		msgDuration := time.Since(msgStart)
		log.Trace().
			Int64("attach_call_id", attachCallID).
			Int("message_index", i).
			Str("message_id", msg.ID.String()).
			Dur("msg_duration", msgDuration).
			Str("next_parent_id", parentID.String()).
			Msg("Message processing complete")
	}

	attachDuration := time.Since(attachStart)
	newNodeCount := len(ct.Nodes)

	log.Trace().
		Int64("attach_call_id", attachCallID).
		Dur("total_duration", attachDuration).
		Int("processed_messages", processedMessages).
		Int("original_node_count", originalNodeCount).
		Int("new_node_count", newNodeCount).
		Int("node_count_increase", newNodeCount-originalNodeCount).
		Str("final_last_id", ct.LastID.String()).
		Msg("TREE ATTACH THREAD COMPLETE")
	return nil
}

// AppendMessages appends a conversation thread to the end of the tree.
// It attaches the thread to the last inserted node in the tree, making it the parent of the thread.
// The messages in the thread are inserted as nodes, extending the parent-child chain.
// Returns an error if duplicate messages are detected or if self-reference cycles would be created.
func (ct *ConversationTree) AppendMessages(thread Conversation) error {
	appendCallID := atomic.AddInt64(&appendMsgCallCounter, 1)
	appendStart := time.Now()

	log.Trace().
		Int64("tree_append_call_id", appendCallID).
		Int("thread_length", len(thread)).
		Str("tree_last_id", ct.LastID.String()).
		Int("tree_node_count", len(ct.Nodes)).
		Msg("TREE APPEND MESSAGES - Delegating to AttachThread")

	if err := ct.AttachThread(ct.LastID, thread); err != nil {
		return errors.Wrap(err, "failed to attach thread during append")
	}

	appendDuration := time.Since(appendStart)
	log.Trace().
		Int64("tree_append_call_id", appendCallID).
		Dur("duration", appendDuration).
		Str("new_last_id", ct.LastID.String()).
		Int("new_node_count", len(ct.Nodes)).
		Msg("TREE APPEND MESSAGES COMPLETE")
	return nil
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
// Includes cycle detection to prevent infinite loops from tree corruption.
func (ct *ConversationTree) GetLeftMostThread(id NodeID) Conversation {
	var thread Conversation
	visited := make(map[NodeID]bool)

	for id != NullNode {
		// Check for cycles - if we've seen this node before, we have a cycle
		if visited[id] {
			log.Trace().
				Str("node_id", id.String()).
				Int("thread_length", len(thread)).
				Msg("CYCLE DETECTED in GetLeftMostThread - breaking to prevent infinite loop")
			break
		}

		node, exists := ct.Nodes[id]
		if !exists {
			log.Trace().
				Str("node_id", id.String()).
				Int("thread_length", len(thread)).
				Msg("Node not found in GetLeftMostThread")
			break
		}

		// Mark this node as visited before processing
		visited[id] = true
		thread = append(thread, node)

		if len(node.Children) > 0 {
			nextID := node.Children[0].ID

			// Additional safety: check if the next node would create an immediate self-cycle
			if nextID == id {
				log.Trace().
					Str("node_id", id.String()).
					Str("next_id", nextID.String()).
					Msg("SELF-REFERENCE DETECTED in GetLeftMostThread - node points to itself")
				break
			}

			log.Trace().
				Str("current_id", id.String()).
				Str("next_id", nextID.String()).
				Int("children_count", len(node.Children)).
				Int("thread_length", len(thread)).
				Msg("GetLeftMostThread traversing to first child")

			id = nextID
		} else {
			log.Trace().
				Str("node_id", id.String()).
				Int("thread_length", len(thread)).
				Msg("GetLeftMostThread reached leaf node")
			id = NullNode
		}
	}

	log.Trace().
		Int("final_thread_length", len(thread)).
		Int("visited_nodes", len(visited)).
		Msg("GetLeftMostThread completed")

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
