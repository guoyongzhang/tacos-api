package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	statsd "github.com/cactus/go-statsd-client/statsd"
	"github.com/mongodb/mongo-go-driver/mongo"
	"github.com/mongodb/mongo-go-driver/mongo/clientopt"
)

type insert struct {
	ID string `json:"id"`
}

var client *mongo.Client
var database *mongo.Database

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func main() {
	var err error

	// create a statsd client
	// The basic client sends one stat per packet (for compatibility).
	statsdClient, err := statsd.NewClient("127.0.0.1:8125", "test-client")
	// handle any errors
	if err != nil {
		log.Fatal(err)
	}
	// make sure to clean up
	defer statsdClient.Close()

	port := ":32001"

	timeout := time.Second * 2
	opt1 := clientopt.ConnectTimeout(timeout)
	opt2 := clientopt.ServerSelectionTimeout(timeout)
	opt3 := clientopt.SocketTimeout(timeout)

	mongoUrl := fmt.Sprintf("mongodb://%s:27017", getEnv("MONGO_HOST", "localhost"))
	fmt.Printf("Connecting to Mongo at %s\n", mongoUrl)
	client, err = mongo.Connect(context.Background(), mongoUrl, opt1, opt2, opt3)
	if err != nil {
		log.Fatal(err)
	}

	database = client.Database("tacos")

	// send a stat every second
	go forever(statsdClient)

	setupStores()
	setupMenuItems()
	setupOrderItems()

	fmt.Printf("Listening (%s)...\n", port)
	http.ListenAndServe(port, nil)
}

// Send a stat
func forever(client statsd.Statter) {
	for {
		time.Sleep(1000 * time.Millisecond)
		x := rand.Intn(10) - 5
		client.Inc("uptime", int64(x), 1.0)
		log.Println("sent uptime stat")
	}
}
