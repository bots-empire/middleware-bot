package redis

import (
	"github.com/BlackRRR/middleware-bot/internal/app/model"
	"github.com/go-redis/redis"
	"log"
	"strconv"
)

var (
	redisDefaultAddr = "127.0.0.1:6379"
	emptyLevelName   = "empty"
)

func StartRedis() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisDefaultAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	return rdb
}

func RdbSetUser(ID int64, level string) {
	userID := userIDToRdb(ID)
	_, err := model.Bot.Rdb.Set(userID, level, 0).Result()
	if err != nil {
		log.Println(err)
	}
}

func userIDToRdb(userID int64) string {
	return "user:" + strconv.FormatInt(userID, 10)
}

func userMsgIDToRdb(userID int64) string {
	return "user_msg:" + strconv.FormatInt(userID, 10)
}

func RdbSetMessageID(ID int64, msgID int) {
	userID := userMsgIDToRdb(ID)
	mID := msgIDToRdb(msgID)
	_, err := model.Bot.Rdb.Set(userID, mID, 0).Result()
	if err != nil {
		log.Println(err)
	}
}

func msgIDToRdb(msgID int) string {
	return strconv.Itoa(msgID)
}

func GetMsgID(userID int64) int {
	uID := userMsgIDToRdb(userID)
	have, err := model.Bot.Rdb.Exists(uID).Result()
	if err != nil {
		log.Println(err)
	}
	if have == 0 {
		return 0
	}

	value, err := model.Bot.Rdb.Get(uID).Result()
	if err != nil {
		log.Println(err)
	}

	parseInt, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parseInt
}

func GetLevel(id int64) string {
	userID := userIDToRdb(id)
	have, err := model.Bot.Rdb.Exists(userID).Result()
	if err != nil {
		log.Println(err)
	}
	if have == 0 {
		return emptyLevelName
	}

	value, err := model.Bot.Rdb.Get(userID).Result()
	if err != nil {
		log.Println(err)
	}
	return value
}
