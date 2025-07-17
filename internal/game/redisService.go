package game

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
	"github.com/thesrcielos/TopTankBattle/pkg/db"
	"github.com/thesrcielos/TopTankBattle/websocket/transport"
)

var subs = make(map[string]*redis.PubSub)

func PublishToRoom(roomID string, payload string) {
	err := db.Rdb.Publish(ctx, "room:"+roomID, payload).Err()
	if err != nil {
		log.Println("Error publishing to room:", err)
	}
}

func SubscribeToRoom(roomID string) error {
	if _, exists := subs[roomID]; exists {
		return nil
	}

	sub := db.Rdb.Subscribe(ctx, "room:"+roomID)

	_, err := sub.Receive(ctx)
	if err != nil {
		return fmt.Errorf("error subscribing to room %s: %w", roomID, err)
	}

	ch := sub.Channel()
	subs[roomID] = sub

	go func() {
		for msg := range ch {
			SendReceivedMessage(msg.Payload)
		}
	}()

	return nil
}

func UnsubscribeFromRoom(roomID string) error {
	sub := subs[roomID]
	if err := sub.Unsubscribe(ctx, "room:"+roomID); err != nil {
		return fmt.Errorf("error unsubscribing from room %s: %w", roomID, err)
	}

	delete(subs, roomID)
	return nil
}

func SendReceivedMessage(messageEncoded string) {
	var message GameMessage
	if err := json.Unmarshal([]byte(messageEncoded), &message); err != nil {
		log.Println("Error decoding message:", err)
		return
	}

	msg := transport.OutgoingMessage{
		Type:    message.Type,
		Payload: message.Payload,
	}

	for _, playerId := range message.Users {
		transport.SendToPlayer(playerId, msg)
	}
}
