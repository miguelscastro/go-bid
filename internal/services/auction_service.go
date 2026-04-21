package services

import (
	"context"
	"errors"
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type MessageKind int

const (
	// Requests
	PlaceBid MessageKind = iota

	// Success
	SuccesfullyPlacedBid

	// Errors
	FailedToPlaceBid

	// Info
	NewBidPlaced
	AuctionFinished
)

type Message struct {
	Message string
	Amount  float64
	Kind    MessageKind
	UserId  uuid.UUID
}

type AuctionLobby struct {
	sync.Mutex // evita race conditions impedindo que duas goroutines afetem dados simultaneamente
	Rooms      map[uuid.UUID]*AuctionRoom
}

type AuctionRoom struct {
	Id         uuid.UUID
	Context    context.Context
	Broadcast  chan Message
	Register   chan *Client
	Unregister chan *Client
	Clients    map[uuid.UUID]*Client

	BidsService BidsService
}

func (r *AuctionRoom) registerClient(c *Client) {
	slog.Info("New user Connected", "Client", c)
	r.Clients[c.UserId] = c
}

func (r *AuctionRoom) unregisterClient(c *Client) {
	slog.Info("User disconnected", "Client", c)
	delete(r.Clients, c.UserId)
}

func (r *AuctionRoom) broadcastMessage(m Message) {
	slog.Info("New message received", "room_id", r.Id, "message", m.Message, "user_id", m.UserId)
	if m.Kind == PlaceBid {
		bid, err := r.BidsService.PlaceBid(r.Context, r.Id, m.UserId, m.Amount)
		if err != nil {
			if errors.Is(err, ErrBidIsToLow) {
				if client, ok := r.Clients[m.UserId]; ok {
					client.Send <- Message{Kind: FailedToPlaceBid, Message: ErrBidIsToLow.Error()}
				}
				return
			}
		}
		if client, ok := r.Clients[m.UserId]; ok {
			client.Send <- Message{Kind: SuccesfullyPlacedBid, Message: "Your bid was succesfully placed"}
		}

		for id, client := range r.Clients {
			newBidMessage := Message{Kind: NewBidPlaced, Message: "A new bid was placed", Amount: bid.BidAmount}
			if id == m.UserId {
				continue
			}
			client.Send <- newBidMessage
		}
	}
}

func (r *AuctionRoom) Run() {
	defer func() {
		close(r.Broadcast)
		close(r.Register)
		close(r.Unregister)
	}()
	for {
		select { // usado para lidar com mais de um channel por vez
		case client := <-r.Register:
			r.registerClient(client)
		case client := <-r.Unregister:
			r.unregisterClient(client)
		case message := <-r.Broadcast:
			r.broadcastMessage(message)
		case <-r.Context.Done():
			slog.Info("auction has ended", "auctionId", r.Id)
			for _, client := range r.Clients {
				client.Send <- Message{Kind: AuctionFinished, Message: "auction has ended"}
			}
			return
		}
	}
}

func NewAuctionRoom(ctx context.Context, id uuid.UUID, BidsService BidsService) *AuctionRoom {
	return &AuctionRoom{
		Id:          id,
		Broadcast:   make(chan Message),
		Register:    make(chan *Client),
		Unregister:  make(chan *Client),
		Context:     ctx,
		BidsService: BidsService,
	}
}

type Client struct {
	Room   *AuctionRoom
	Conn   *websocket.Conn
	Send   chan Message
	UserId uuid.UUID
}

func NewClient(room *AuctionRoom, conn *websocket.Conn, userId uuid.UUID) *Client {
	return &Client{
		Room: room,
		Conn: conn,
		// unbuffered channel -> recebe uma msg por vez
		// buffered channel -> recebe o N de msg especificado -> make(chan Message, 512)
		Send:   make(chan Message, 512),
		UserId: userId,
	}
}
