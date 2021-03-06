package testutil

import (
	"github.com/satori/go.uuid"
	"github.com/aaronaaeng/chat.connor.fun/model"
	"github.com/aaronaaeng/chat.connor.fun/db"
)

type MockRoomsRepository struct {
	Rooms map[uuid.UUID]model.ChatRoom
}

func NewMockRoomsRepository() *MockRoomsRepository {
	return &MockRoomsRepository{map[uuid.UUID]model.ChatRoom{}}
}

func (r *MockRoomsRepository) NewFromSource(source db.DataSource) db.RoomsRepository {
	return r
}

func (r MockRoomsRepository) Add(room *model.ChatRoom) error {
	r.Rooms[room.Id] = *room
	return nil
}

func (r MockRoomsRepository) GetById(id uuid.UUID) (*model.ChatRoom, error) {
	room, ok := r.Rooms[id]
	if !ok {
		return nil, nil
	}
	retRoom := room
	return &retRoom, nil
}

func (r MockRoomsRepository) GetByName(name string) (*model.ChatRoom, error) {
	for room := range r.Rooms {
		if r.Rooms[room].Name == name {
			toReturn := r.Rooms[room]
			return &toReturn, nil
		}
	}
	return nil, nil
}

func (r MockRoomsRepository) GetWithinArea(area *model.GeoArea) ([]*model.RelativeRoom, error) {
	return nil, nil //this will be very annoying to mock properly
}