package main

import (
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/google/uuid"
)

var lettersRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func main() {

	redisHost := os.Getenv("REDIS_URL")
	redisHost = strings.TrimSpace(redisHost)
	if redisHost == "" {
		redisHost = "localhost:6379"
	}

	// エラー情報を入れるChannel
	c := make(chan string)
	for i := 0; i < 10; i++ {
		if i%10 == 0 {
			go addElementWithTTL(c, redisHost)
		} else {
			go addEternalElement(c, redisHost)
		}
	}
	defer close(c)

	// TODO: 5秒でTimeout
	timeout := time.After(5 * time.Second)
	for {
		// TimeoutもChannelらしい
		select {
		case e, ok := <-c:
			if !ok {
				// closed
				return
			}
			fmt.Println(e)
		case <-timeout:
			fmt.Println("Finished")
			return
		}
	}
}

// 延々とアイテムを詰め続ける
func addEternalElement(c chan string, redisHost string) {

	// Open Database index 0
	conn, err := redis.Dial("tcp", redisHost, redis.DialDatabase(0))
	if err != nil {
		// エラー通知
		c <- err.Error()
		return
	}
	defer conn.Close()

	for {
		// Add element
		u, err := uuid.NewRandom()
		key := u.String()
		if err != nil {
			c <- err.Error()
		}

		// TTLなしで1024文字を登録
		_, err = conn.Do("SET", key, randomStringRunes(1024))

		if err != nil {
			// Errorが出たらChannelにエラーを突っ込んで処理をforから脱出
			c <- err.Error()
			break
		} else {
			// 10ms Sleep
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// TTL is time to live for redis item
const TTL = 18000

func addElementWithTTL(c chan string, redisHost string) {

	// Open Database index 0
	conn, err := redis.Dial("tcp", redisHost, redis.DialDatabase(0))

	// Add element
	u, err := uuid.NewRandom()
	key := u.String()
	if err != nil {
		c <- err.Error()
	}

	if err != nil {
		// エラー通知
		c <- err.Error()
		return
	}
	defer conn.Close()

	for {
		// 1024文字のランダム文字列を指定のTTLで登録
		_, err = conn.Do("SET", key, randomStringRunes(1024), "EX", TTL)

		if err != nil {
			// Errorが出たらChannelにエラーを突っ込んで処理をforから脱出
			c <- err.Error()
			break
		} else {
			// 10ms Sleep
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func setAndExpire(conn redis.Conn) string {
	// Add element
	key := uuid.New().String

	// 1024文字のランダム文字列を60secのTTLで登録
	_, err := conn.Do("SET", key, randomStringRunes(1024), "EX", 60)
	if err != nil {
		panic(err)
	}

	// Get Stored Item
	s, err := redis.String(conn.Do("GET", key))
	if err != nil {
		panic(err)
	}
	return s
}

// 指定した数のランダム文字列を生成
func randomStringRunes(n int) string {
	rand.Seed(time.Now().UnixNano()) // randを初期化
	b := make([]rune, n)             // 最大長のsliceを定義
	for i := range b {
		b[i] = lettersRunes[rand.Intn(len(lettersRunes))]
	}
	return string(b)
}
