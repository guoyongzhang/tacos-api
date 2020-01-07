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

type menuItem struct {
	ID    objectid.ObjectID `bson:"_id"`
	Key   string            `json:"key"`
	Type  string            `json:"type"`
	Store string            `json:"store"`
	Name  string            `json:"name"`
	Slug  string            `json:"slug"`
	Descr string            `json:"descr"`
	Price string            `json:"price"`
}

var menuItemsColl *mongo.Collection

func handleMenuItems(res http.ResponseWriter, req *http.Request) {
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
	case "GET": // list items for store, id in path
		itemID := strings.TrimPrefix(req.URL.Path, "/api/v1/menu/")
		log.Printf("get param: %s", itemID)
		oid, err := objectid.FromHex(itemID)
		if err != nil {
			httpError(err.Error())
			return
		}
		filter := bson.NewDocument(bson.EC.ObjectID("store", oid))
		cur, err := menuItemsColl.Find(context.Background(), filter)
		if err != nil {
			httpError(err.Error())
			return
		}
		defer cur.Close(context.Background())
		list := make([]menuItem, 0)
		for cur.Next(context.Background()) {
			var item menuItem
			err := cur.Decode(&item)
			if err != nil {
				httpError(err.Error())
				return
			}
			item.Key = item.ID.Hex()
			fmt.Printf("item: %+v\n", item)
			fmt.Printf("name = %+v\n", item.Name)
			list = append(list, item)
		}
		res.Header().Set("Content-Type", "application/json")
		json.NewEncoder(res).Encode(list)

	case "PUT": // add item
		// todo: make sure everything is specified and valid
		decoder := json.NewDecoder(req.Body)
		defer req.Body.Close()
		var item menuItem
		err := decoder.Decode(&item)
		if err != nil {
			httpError(err.Error())
			return
		}
		fmt.Printf("item: %+v\n", item)
		inserts := make([]*bson.Element, 0)
		if item.Type != "" {
			inserts = append(inserts, bson.EC.String("type", item.Type))
			switch item.Type {
			case "base", "filling", "topping":
				break
			default:
				httpError("Type must be one of base, filling, topping")
				return
			}
		} else {
			httpError("Type is required")
			return
		}
		if item.Store != "" {
			oid, err := objectid.FromHex(item.Store)
			if err != nil {
				httpError(err.Error())
				return
			}
			inserts = append(inserts, bson.EC.ObjectID("store", oid))
		}
		if item.Name != "" {
			inserts = append(inserts, bson.EC.String("name", item.Name))
		}
		if item.Slug != "" {
			inserts = append(inserts, bson.EC.String("slug", item.Slug))
		}
		if item.Descr != "" {
			inserts = append(inserts, bson.EC.String("descr", item.Descr))
		}
		if item.Price != "" {
			inserts = append(inserts, bson.EC.String("price", item.Price))
		}
		fmt.Printf("inserts: %+v\n", inserts)
		inserter := bson.NewDocument()
		for _, update := range inserts {
			inserter.Append(update)
		}
		fmt.Printf("inserter: %+v\n", inserter)
		result, err := menuItemsColl.InsertOne(context.Background(), inserter, nil)
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

	case "PATCH": // edit item, id in path
		itemID := strings.TrimPrefix(req.URL.Path, "/api/v1/menu/")
		log.Printf("patch param: %s", itemID)
		oid, err := objectid.FromHex(itemID)
		if err != nil {
			httpError(err.Error())
			return
		}
		updater := bson.NewDocument(bson.EC.ObjectID("_id", oid))
		decoder := json.NewDecoder(req.Body)
		defer req.Body.Close()
		var item menuItem
		err = decoder.Decode(&item)
		if err != nil {
			httpError(err.Error())
			return
		}
		fmt.Printf("updater: %+v\n", updater)
		updates := make([]*bson.Element, 0)
		if item.Type != "" {
			httpError("Item type may not be changed")
		}
		if item.Store != "" {
			httpError("Item store may not be changed")
		}
		if item.Name != "" {
			updates = append(updates, bson.EC.String("name", item.Name))
		}
		if item.Slug != "" {
			updates = append(updates, bson.EC.String("slug", item.Slug))
		}
		if item.Descr != "" {
			updates = append(updates, bson.EC.String("descr", item.Descr))
		}
		if item.Price != "" {
			updates = append(updates, bson.EC.String("price", item.Price))
		}
		fmt.Printf("updates: %+v\n", updates)
		subdoc := bson.NewDocument()
		for _, update := range updates {
			subdoc.Append(update)
		}
		setter := bson.NewDocument(bson.EC.SubDocument("$set", subdoc))
		fmt.Printf("setter: %+v\n", setter)
		result, err := menuItemsColl.UpdateOne(context.Background(), updater, setter, nil)
		if err != nil {
			httpError(err.Error())
			return
		}
		fmt.Printf("result: %+v\n", result)
		res.Header().Set("Content-Type", "application/json")
		json.NewEncoder(res).Encode(result)

	case "DELETE": // delete item, id in path
		itemID := strings.TrimPrefix(req.URL.Path, "/api/v1/menu/")
		log.Printf("patch param: %s", itemID)
		oid, err := objectid.FromHex(itemID)
		if err != nil {
			httpError(err.Error())
			return
		}
		deleter := bson.NewDocument(bson.EC.ObjectID("_id", oid))
		result, err := menuItemsColl.DeleteOne(context.Background(), deleter, nil)
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

func setupMenuItems() {
	menuItemsColl = database.Collection("menu_items")

	http.HandleFunc("/api/v1/menu", handleMenuItems)
	http.HandleFunc("/api/v1/menu/", handleMenuItems)
}
