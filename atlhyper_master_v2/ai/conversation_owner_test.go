package ai

import (
	"context"
	"errors"
	"testing"

	"AtlHyper/atlhyper_master_v2/database"
)

type fakeConvRepo struct {
	database.AIConversationRepository
	convs   map[int64]*database.AIConversation
	deleted []int64
}

func (r *fakeConvRepo) GetByID(_ context.Context, id int64) (*database.AIConversation, error) {
	return r.convs[id], nil
}
func (r *fakeConvRepo) Delete(_ context.Context, id int64) error {
	r.deleted = append(r.deleted, id)
	return nil
}

type fakeMsgRepo struct {
	database.AIMessageRepository
	msgs    map[int64][]*database.AIMessage
	deleted []int64
}

func (r *fakeMsgRepo) ListByConversation(_ context.Context, id int64) ([]*database.AIMessage, error) {
	return r.msgs[id], nil
}
func (r *fakeMsgRepo) DeleteByConversation(_ context.Context, id int64) error {
	r.deleted = append(r.deleted, id)
	return nil
}

const (
	ownerID  int64 = 1
	otherID  int64 = 2
	convOfA  int64 = 10
	missing  int64 = 99
	ownerMsg       = "owner's question"
)

func newOwnerTestService() (*aiServiceImpl, *fakeConvRepo, *fakeMsgRepo) {
	convs := &fakeConvRepo{convs: map[int64]*database.AIConversation{convOfA: {ID: convOfA, UserID: ownerID}}}
	msgs := &fakeMsgRepo{msgs: map[int64][]*database.AIMessage{convOfA: {{ConversationID: convOfA, Role: "user", Content: ownerMsg}}}}
	return &aiServiceImpl{convRepo: convs, msgRepo: msgs}, convs, msgs
}

// 会话只属于创建者：别人读、续聊、删除一律当作不存在
func TestConversation_OnlyOwnerCanRead(t *testing.T) {
	s, _, _ := newOwnerTestService()

	msgs, err := s.GetMessages(context.Background(), ownerID, convOfA)
	if err != nil || len(msgs) != 1 || msgs[0].Content != ownerMsg {
		t.Fatalf("创建者读自己的会话：msgs=%v err=%v", msgs, err)
	}
	for _, c := range []struct {
		user, conv int64
	}{{otherID, convOfA}, {ownerID, missing}} {
		if _, err := s.GetMessages(context.Background(), c.user, c.conv); !errors.Is(err, ErrConversationNotFound) {
			t.Errorf("user %d 读会话 %d：err=%v, want ErrConversationNotFound", c.user, c.conv, err)
		}
	}
}

func TestConversation_OnlyOwnerCanContinue(t *testing.T) {
	s, _, _ := newOwnerTestService()
	_, err := s.Chat(context.Background(), &ChatRequest{ConversationID: convOfA, UserID: otherID, Message: "hi"})
	if !errors.Is(err, ErrConversationNotFound) {
		t.Errorf("别人续聊：err=%v, want ErrConversationNotFound", err)
	}
}

func TestConversation_OnlyOwnerCanDelete(t *testing.T) {
	s, convs, msgs := newOwnerTestService()
	if err := s.DeleteConversation(context.Background(), otherID, convOfA); !errors.Is(err, ErrConversationNotFound) {
		t.Errorf("别人删除：err=%v, want ErrConversationNotFound", err)
	}
	if len(convs.deleted) != 0 || len(msgs.deleted) != 0 {
		t.Fatalf("别人删除不得动数据：convs=%v msgs=%v", convs.deleted, msgs.deleted)
	}
	if err := s.DeleteConversation(context.Background(), ownerID, convOfA); err != nil {
		t.Fatalf("创建者删除：%v", err)
	}
	if len(convs.deleted) != 1 || len(msgs.deleted) != 1 {
		t.Errorf("创建者删除应删掉会话与消息：convs=%v msgs=%v", convs.deleted, msgs.deleted)
	}
}
