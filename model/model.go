package model

type Type string

const (
	WaitingSticker   Type = "WaitingSticker"
	AcceptingTags    Type = "AcceptingTags"
	AcceptingSticker Type = "AcceptingSticker"
)

type State struct {
	Type      Type
	StickerId string
	Tag       string
}

type StickerR struct {
	UserId    int      `bson:"userId"`
	RecordId  int      `bson:"recordId"`
	StickerId string   `bson:"stickerId"`
	Tags      []string `bson:"tags"`
}
