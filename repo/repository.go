package repo

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"log"
	"stkrz/model"
	"time"
)

const MONGO_URL = "mongodb://localhost"
const DB_NAME = "stkrz"

func init() {
	sess := GetSession()
	defer sess.Close()

	rows, err := GetSeq(sess).Find(bson.M{"seq": "recordId"}).Count()

	if err != nil {
		log.Println("Error initializing recordId sequence!!!")
		return
	}

	if rows == 0 {
		GetSeq(sess).Insert(bson.M{"seq": "recordId"})
		log.Println("Created recordId sequence!")
	}

	GetPersonal(sess).EnsureIndex(
		mgo.Index{
			Key:    []string{"userId", "stickerId"},
			Unique: true,
		},
	)

	GetPersonal(sess).EnsureIndex(
		mgo.Index{
			Key: []string{"stickerId"},
		},
	)
}

func GetSession() *mgo.Session {
	session, err := mgo.Dial(MONGO_URL)
	if err != nil {
		time.Sleep(50 * time.Millisecond)
		session, err = mgo.Dial(MONGO_URL)
		if err != nil {
			panic(err)
		}
	}
	session.SetMode(mgo.Monotonic, true)
	return session
}

func GetPersonal(sess *mgo.Session) *mgo.Collection {
	return sess.DB(DB_NAME).C("personal")
}

func GetSeq(sess *mgo.Session) *mgo.Collection {
	return sess.DB(DB_NAME).C("seq")
}

func GetNewId() int {
	sess := GetSession()
	defer sess.Close()

	type Ix struct {
		Ix int `bson:"ix"`
	}
	var ix Ix
	change := mgo.Change{
		Update:    bson.M{"$inc": bson.M{"ix": 1}},
		ReturnNew: true,
	}
	_, err := GetSeq(sess).Find(bson.M{"seq": "recordId"}).Apply(change, &ix)

	if err != nil {
		return 1
	} else {
		return ix.Ix
	}
}

func FindByQuery(userId int, query string) []model.StickerR {
	sess := GetSession()
	defer sess.Close()
	var stickers []model.StickerR
	GetPersonal(sess).Find(bson.M{"userId": userId, "tags": bson.M{"$all": []string{query}}}).
		Limit(10).
		All(&stickers)
	return stickers
}

func ClearTags(stickerId string, userId int) {
	sess := GetSession()
	defer sess.Close()
	GetPersonal(sess).Update(
		bson.M{"stickerId": stickerId, "userId": userId},
		bson.M{"$unset": bson.M{"tags": 1}},
	)
}

func SetTags(stickerId string, userId int, newTags []string) error {
	sess := GetSession()
	defer sess.Close()
	return GetPersonal(sess).Update(
		bson.M{"stickerId": stickerId, "userId": userId},
		bson.M{"$set": bson.M{"tags": newTags}},
	)
}
func FindSticker(stickerId string, userId int) (model.StickerR, error) {
	var s model.StickerR
	sess := GetSession()
	defer sess.Close()
	err := GetPersonal(sess).Find(bson.M{"stickerId": stickerId, "userId": userId}).One(&s)
	return s, err
}
