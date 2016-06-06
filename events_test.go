package gmunch

import (
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
)

type coolEventData struct {
	CustomerId  string
	UserId      int
	Permissions []string
	Attributes  map[string]interface{}
}

func TestFieldsMap(t *testing.T) {
	assert := assert.New(t)
	coolEvent := &Event{Name: "cool"}
	coolEvent.EncodeData(&coolEventData{
		CustomerId:  "custy-asdf",
		UserId:      7,
		Permissions: []string{"read", "write", "admin"},
		Attributes: map[string]interface{}{
			"cool": true,
		},
	})

	// send event through marshaling and back to ensure consistency
	pbdata, err := proto.Marshal(coolEvent)
	if err != nil {
		t.Fatal(err)
	}

	event := &Event{}
	err = proto.Unmarshal(pbdata, event)
	if err != nil {
		t.Fatal(err)
	}

	coolData := &coolEventData{}
	err = event.Decoder().Decode(coolData)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal("custy-asdf", coolData.CustomerId)
	assert.Equal(7, coolData.UserId)
	assert.Equal([]string{"read", "write", "admin"}, coolData.Permissions)
	assert.True(coolData.Attributes["cool"].(bool))
}
