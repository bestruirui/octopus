package relay

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lingyuins/octopus/internal/server/resp"
)

var (
	relayStreamSessionTTL       = 30 * time.Minute
	relayStreamSessionMaxEvents = 4096
	relayStreamSessionMaxBytes  = 16 << 20

	errRelayConversationBusy    = errors.New("conversation already has an active generation")
	errRelayReplayWindowExpired = errors.New("relay stream replay window expired")
)

type relayStreamEvent struct {
	Sequence int64
	Payload  []byte
}

type relayStreamSession struct {
	store             *relayStreamSessionStore
	key               string
	conversationID    string
	conversationScope string
	requestHash       uint64
	createdAt         time.Time

	mu               sync.RWMutex
	updatedAt        time.Time
	done             bool
	err              error
	nextSeq          int64
	droppedBeforeSeq int64
	bufferBytes      int
	events           []relayStreamEvent
	subscribers      map[chan struct{}]struct{}
}

type relayStreamSessionStore struct {
	mu                   sync.RWMutex
	byKey                map[string]*relayStreamSession
	activeByConversation map[string]string
}

var relayStreamSessions = relayStreamSessionStore{
	byKey:                make(map[string]*relayStreamSession),
	activeByConversation: make(map[string]string),
}

func buildRelayStreamSessionKey(conversationID string, requestHash uint64) string {
	return strings.TrimSpace(conversationID) + ":" + strconv.FormatUint(requestHash, 16)
}

func buildRelayConversationScope(conversationID string, apiKeyID int) string {
	return strconv.Itoa(apiKeyID) + ":" + strings.TrimSpace(conversationID)
}

func acquireRelayStreamSession(conversationID string, apiKeyID int, requestHash uint64) (*relayStreamSession, bool, error) {
	conversationID = strings.TrimSpace(conversationID)
	if conversationID == "" || requestHash == 0 {
		return nil, false, nil
	}

	now := time.Now()
	store := &relayStreamSessions
	conversationScope := buildRelayConversationScope(conversationID, apiKeyID)
	key := buildRelayStreamSessionKey(conversationScope, requestHash)

	store.mu.Lock()
	defer store.mu.Unlock()

	store.cleanupLocked(now)

	if session, ok := store.byKey[key]; ok {
		return session, false, nil
	}

	if activeKey, ok := store.activeByConversation[conversationScope]; ok && activeKey != key {
		if activeSession, exists := store.byKey[activeKey]; exists && !activeSession.isDoneLocked() {
			return nil, false, errRelayConversationBusy
		}
		delete(store.activeByConversation, conversationScope)
	}

	session := &relayStreamSession{
		store:             store,
		key:               key,
		conversationID:    conversationID,
		conversationScope: conversationScope,
		requestHash:       requestHash,
		createdAt:         now,
		updatedAt:         now,
		subscribers:       make(map[chan struct{}]struct{}),
	}
	store.byKey[key] = session
	store.activeByConversation[conversationScope] = key
	return session, true, nil
}

func (s *relayStreamSessionStore) cleanupLocked(now time.Time) {
	for key, session := range s.byKey {
		session.mu.RLock()
		done := session.done
		updatedAt := session.updatedAt
		conversationScope := session.conversationScope
		sessionKey := session.key
		session.mu.RUnlock()

		if !done {
			continue
		}
		if now.Sub(updatedAt) < relayStreamSessionTTL {
			continue
		}

		delete(s.byKey, key)
		if activeKey, ok := s.activeByConversation[conversationScope]; ok && activeKey == sessionKey {
			delete(s.activeByConversation, conversationScope)
		}
	}
}

func (s *relayStreamSessionStore) removeIfExpired(key string, conversationScope string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.byKey[key]
	if !ok {
		return
	}

	session.mu.RLock()
	done := session.done
	updatedAt := session.updatedAt
	sessionKey := session.key
	sessionScope := session.conversationScope
	session.mu.RUnlock()

	if !done || sessionScope != conversationScope || time.Since(updatedAt) < relayStreamSessionTTL {
		return
	}

	delete(s.byKey, key)
	if activeKey, ok := s.activeByConversation[sessionScope]; ok && activeKey == sessionKey {
		delete(s.activeByConversation, sessionScope)
	}
}

func (s *relayStreamSession) isDoneLocked() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.done
}

func (s *relayStreamSession) IsDone() bool {
	if s == nil {
		return true
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.done
}

func (s *relayStreamSession) AddPayload(payload []byte) []relayStreamEvent {
	if s == nil || len(payload) == 0 {
		return nil
	}

	frames := splitRelaySSEPayload(payload)
	if len(frames) == 0 {
		return nil
	}

	events := make([]relayStreamEvent, 0, len(frames))

	s.mu.Lock()
	for _, frame := range frames {
		s.nextSeq++
		event := relayStreamEvent{
			Sequence: s.nextSeq,
			Payload:  frame,
		}
		s.events = append(s.events, event)
		s.bufferBytes += len(frame)
		events = append(events, event)
	}
	s.trimEventsLocked()
	s.updatedAt = time.Now()

	subscribers := make([]chan struct{}, 0, len(s.subscribers))
	for ch := range s.subscribers {
		subscribers = append(subscribers, ch)
	}
	s.mu.Unlock()

	for _, ch := range subscribers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}

	return events
}

func (s *relayStreamSession) trimEventsLocked() {
	for len(s.events) > 0 {
		tooManyEvents := relayStreamSessionMaxEvents > 0 && len(s.events) > relayStreamSessionMaxEvents
		tooManyBytes := relayStreamSessionMaxBytes > 0 && s.bufferBytes > relayStreamSessionMaxBytes && len(s.events) > 1
		if !tooManyEvents && !tooManyBytes {
			return
		}

		dropped := s.events[0]
		s.droppedBeforeSeq = dropped.Sequence
		s.bufferBytes -= len(dropped.Payload)
		if s.bufferBytes < 0 {
			s.bufferBytes = 0
		}
		s.events[0].Payload = nil
		s.events = s.events[1:]
	}
}

func (s *relayStreamSession) Snapshot(afterSeq int64) ([]relayStreamEvent, bool, error) {
	if s == nil {
		return nil, true, nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if afterSeq < s.droppedBeforeSeq {
		return nil, s.done, errRelayReplayWindowExpired
	}

	idx := 0
	for idx < len(s.events) && s.events[idx].Sequence <= afterSeq {
		idx++
	}

	events := make([]relayStreamEvent, 0, len(s.events)-idx)
	for ; idx < len(s.events); idx++ {
		event := s.events[idx]
		event.Payload = append([]byte(nil), event.Payload...)
		events = append(events, event)
	}

	return events, s.done, s.err
}

func (s *relayStreamSession) Subscribe() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	if s == nil {
		close(ch)
		return ch, func() {}
	}

	s.mu.Lock()
	if s.done {
		s.mu.Unlock()
		close(ch)
		return ch, func() {}
	}
	s.subscribers[ch] = struct{}{}
	s.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			s.mu.Lock()
			if _, ok := s.subscribers[ch]; ok {
				delete(s.subscribers, ch)
				close(ch)
			}
			s.mu.Unlock()
		})
	}

	return ch, unsubscribe
}

func (s *relayStreamSession) Finish(err error) {
	if s == nil {
		return
	}

	s.mu.Lock()
	if s.done {
		s.mu.Unlock()
		return
	}
	s.done = true
	s.err = err
	s.updatedAt = time.Now()

	subscribers := make([]chan struct{}, 0, len(s.subscribers))
	for ch := range s.subscribers {
		subscribers = append(subscribers, ch)
	}
	s.subscribers = make(map[chan struct{}]struct{})
	s.mu.Unlock()

	s.store.mu.Lock()
	if activeKey, ok := s.store.activeByConversation[s.conversationScope]; ok && activeKey == s.key {
		delete(s.store.activeByConversation, s.conversationScope)
	}
	s.store.mu.Unlock()

	if relayStreamSessionTTL > 0 {
		time.AfterFunc(relayStreamSessionTTL, func() {
			s.store.removeIfExpired(s.key, s.conversationScope)
		})
	}

	for _, ch := range subscribers {
		close(ch)
	}
}

func splitRelaySSEPayload(payload []byte) [][]byte {
	trimmed := bytes.TrimLeft(payload, "\r\n")
	if len(trimmed) == 0 {
		return nil
	}

	parts := bytes.Split(trimmed, []byte("\n\n"))
	frames := make([][]byte, 0, len(parts))
	for _, part := range parts {
		frame := bytes.TrimLeft(part, "\r\n")
		if len(bytes.TrimSpace(frame)) == 0 {
			continue
		}
		cloned := append([]byte(nil), frame...)
		if !bytes.HasSuffix(cloned, []byte("\n\n")) {
			cloned = append(cloned, '\n', '\n')
		}
		frames = append(frames, cloned)
	}
	return frames
}

func formatRelaySSEEvent(sequence int64, payload []byte) []byte {
	frame := make([]byte, 0, len(payload)+32)
	frame = append(frame, []byte("id: "+strconv.FormatInt(sequence, 10)+"\n")...)
	frame = append(frame, payload...)
	if !bytes.HasSuffix(frame, []byte("\n\n")) {
		frame = append(frame, '\n', '\n')
	}
	return frame
}

func serveRelayStreamSession(c *gin.Context, req *relayRequest) {
	if req == nil || req.streamSession == nil {
		resp.Error(c, http.StatusBadRequest, "missing relay stream session")
		return
	}

	clientCtx := c.Request.Context()
	lastSeq := req.internalRequest.ResumeFromEventID
	headersWritten := false

	writeHeaders := func() {
		if headersWritten {
			return
		}
		headersWritten = true
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		c.Header("X-Conversation-ID", req.internalRequest.ConversationID)
	}

	writeEvents := func(events []relayStreamEvent) bool {
		for _, event := range events {
			writeHeaders()
			if _, err := c.Writer.Write(formatRelaySSEEvent(event.Sequence, event.Payload)); err != nil {
				return false
			}
			c.Writer.Flush()
			lastSeq = event.Sequence
		}
		return true
	}

	sub, unsubscribe := req.streamSession.Subscribe()
	defer unsubscribe()

	for {
		events, done, sessionErr := req.streamSession.Snapshot(lastSeq)
		if errors.Is(sessionErr, errRelayReplayWindowExpired) {
			if !headersWritten {
				resp.Error(c, http.StatusConflict, sessionErr.Error())
			}
			return
		}
		if len(events) > 0 {
			if !writeEvents(events) {
				return
			}
		}

		if done {
			if !headersWritten && sessionErr != nil {
				statusCode := http.StatusBadGateway
				if errors.Is(sessionErr, context.DeadlineExceeded) {
					statusCode = http.StatusGatewayTimeout
				}
				resp.Error(c, statusCode, sessionErr.Error())
				return
			}
			return
		}

		select {
		case <-clientCtx.Done():
			return
		case _, ok := <-sub:
			if !ok {
				continue
			}
		}
	}
}
