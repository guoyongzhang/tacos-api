package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"runtime"
	"strings"

	"github.com/mongodb/mongo-go-driver/bson"
	"github.com/mongodb/mongo-go-driver/bson/objectid"
	"github.com/mongodb/mongo-go-driver/mongo"
)

// Store is a store
type Store struct {
	ID      objectid.ObjectID `bson:"_id"`
	IDStr   string            `json:"id"`
	Type    string            `json:"type"`
	Name    string            `json:"name"`
	Address string            `json:"address"`
	City    string            `json:"city"`
	State   string            `json:"state"`
	Zip     string            `json:"zip"`
}

var storesColl *mongo.Collection

func handleStores(res http.ResponseWriter, req *http.Request) {
	httpError := func(msg string) {
		// todo: restrict this to debug only
		_, callerFile, callerLine, ok := runtime.Caller(1)
		if ok {
			split := strings.Split(path.Base(callerFile), ".")
			msg += fmt.Sprintf(" at %s:%d", split[0], callerLine)
		}
		http.Error(res, msg, 500)
	}

	switch req.Method {
	case "GET": // list stores unless ID is specified
		storeID := strings.TrimPrefix(req.URL.Path, "/api/v1/stores/")
		if storeID == "/api/v1/stores" {
			cur, err := storesColl.Find(context.Background(), nil)
			if err != nil {
				httpError(err.Error())
				return
			}
			defer cur.Close(context.Background())
			list := make([]Store, 0)
			for cur.Next(context.Background()) {
				var store Store
				err := cur.Decode(&store)
				if err != nil {
					httpError(err.Error())
					return
				}
				store.IDStr = store.ID.Hex()
				list = append(list, store)
			}
			res.Header().Set("Content-Type", "application/json")
			json.NewEncoder(res).Encode(list)
		} else {
			// todo: return an error if the store doesn't exist?
			log.Printf("patch param: %s", storeID)
			oid, err := objectid.FromHex(storeID)
			if err != nil {
				httpError(err.Error())
				return
			}
			filter := bson.NewDocument(bson.EC.ObjectID("_id", oid))
			cur, err := storesColl.Find(context.Background(), filter)
			if err != nil {
				httpError(err.Error())
				return
			}
			defer cur.Close(context.Background())
			var store Store
			if cur.Next(context.Background()) {
				err := cur.Decode(&store)
				if err != nil {
					httpError(err.Error())
					return
				}
				store.IDStr = store.ID.Hex()
			}
			res.Header().Set("Content-Type", "application/json")
			json.NewEncoder(res).Encode(store)
		}

	case "PUT": // add store
		// todo: make sure everything is specified and valid
		decoder := json.NewDecoder(req.Body)
		defer req.Body.Close()
		var store Store
		err := decoder.Decode(&store)
		if err != nil {
			httpError(err.Error())
			return
		}
		fmt.Printf("store: %+v\n", store)
		inserts := make([]*bson.Element, 0)
		if store.Type != "" {
			inserts = append(inserts, bson.EC.String("type", store.Type))
			switch store.Type {
			case "tacos", "icecream", "other":
				break
			default:
				httpError("Type must be one of tacos, icecream, other")
				return
			}
		} else {
			httpError("Type is required")
			return
		}
		if store.Name != "" {
			inserts = append(inserts, bson.EC.String("name", store.Name))
		}
		if store.Address != "" {
			inserts = append(inserts, bson.EC.String("address", store.Address))
		}
		if store.City != "" {
			inserts = append(inserts, bson.EC.String("city", store.City))
		}
		if store.State != "" {
			inserts = append(inserts, bson.EC.String("state", store.State))
		}
		if store.Zip != "" {
			inserts = append(inserts, bson.EC.String("zip", store.Zip))
		}
		fmt.Printf("inserts: %+v\n", inserts)
		inserter := bson.NewDocument()
		for _, update := range inserts {
			inserter.Append(update)
		}
		fmt.Printf("inserter: %+v\n", inserter)
		result, err := storesColl.InsertOne(context.Background(), inserter, nil)
		if err != nil {
			httpError(err.Error())
			return
		}
		fmt.Printf("result: %+v\n", result)
		res.Header().Set("Content-Type", "application/json")
		if oid, ok := result.InsertedID.(objectid.ObjectID); ok {
			json.NewEncoder(res).Encode(insert{oid.Hex()})
		} else {
			json.NewEncoder(res).Encode(result)
		}

	case "PATCH": // edit store, id in path
		storeID := strings.TrimPrefix(req.URL.Path, "/api/v1/stores/")
		log.Printf("patch param: %s", storeID)
		oid, err := objectid.FromHex(storeID)
		if err != nil {
			httpError(err.Error())
			return
		}
		updater := bson.NewDocument(bson.EC.ObjectID("_id", oid))
		decoder := json.NewDecoder(req.Body)
		defer req.Body.Close()
		var store Store
		err = decoder.Decode(&store)
		if err != nil {
			httpError(err.Error())
			return
		}
		fmt.Printf("updater: %+v\n", updater)
		updates := make([]*bson.Element, 0)
		if store.Type != "" {
			httpError("Store type may not be changed")
		}
		if store.Name != "" {
			updates = append(updates, bson.EC.String("name", store.Name))
		}
		if store.Address != "" {
			updates = append(updates, bson.EC.String("address", store.Address))
		}
		if store.City != "" {
			updates = append(updates, bson.EC.String("city", store.City))
		}
		if store.State != "" {
			updates = append(updates, bson.EC.String("state", store.State))
		}
		if store.Zip != "" {
			updates = append(updates, bson.EC.String("zip", store.Zip))
		}
		fmt.Printf("updates: %+v\n", updates)
		subdoc := bson.NewDocument()
		for _, update := range updates {
			subdoc.Append(update)
		}
		setter := bson.NewDocument(bson.EC.SubDocument("$set", subdoc))
		fmt.Printf("setter: %+v\n", setter)
		result, err := storesColl.UpdateOne(context.Background(), updater, setter, nil)
		if err != nil {
			httpError(err.Error())
			return
		}
		fmt.Printf("result: %+v\n", result)
		res.Header().Set("Content-Type", "application/json")
		json.NewEncoder(res).Encode(result)

	case "DELETE": // delete store, id in path
		storeID := strings.TrimPrefix(req.URL.Path, "/api/v1/stores/")
		log.Printf("delete param: %s", storeID)
		oid, err := objectid.FromHex(storeID)
		if err != nil {
			httpError(err.Error())
			return
		}
		deleter := bson.NewDocument(bson.EC.ObjectID("_id", oid))
		result, err := storesColl.DeleteOne(context.Background(), deleter, nil)
		if err != nil {
			httpError(err.Error())
			return
		}
		fmt.Printf("result: %+v\n", result)
		res.Header().Set("Content-Type", "application/json")
		json.NewEncoder(res).Encode(result)

	default:
		for key, header := range req.Header {
			fmt.Println("req #", key, ":", header)
		}
		fmt.Println("req method:", req.Method)

		decoder := json.NewDecoder(req.Body)
		defer req.Body.Close()
		var reqBody struct {
			ID   string
			Name string
		}
		err := decoder.Decode(&reqBody)
		if err != nil {
			httpError(err.Error())
			return
		}
		fmt.Printf("req body: %+v\n", reqBody)

		httpError(fmt.Sprintf("Unexpected method %s", req.Method))
	}
}

func setupStores() {
	storesColl = database.Collection("stores")

	http.HandleFunc("/api/v1/stores", handleStores)
	http.HandleFunc("/api/v1/stores/", handleStores)
}
