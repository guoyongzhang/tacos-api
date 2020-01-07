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

type orderTrans struct {
	Cust    string `json:"cust"`
	Store   string `json:"store"`
	Started int    `json:"started"` // timestamp
	Done    int    `json:"done"`    // timestamp
}

type orderItem struct {
	Order string `json:"order"`
	Item  string `json:"item"`
	Count int    `json:"count"`
}

var ordersColl *mongo.Collection
var orderItemsColl *mongo.Collection

func handleOrderItems(res http.ResponseWriter, req *http.Request) {
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
	case "POST":
		if req.URL.Path == "/api/v1/order" {
			// todo: make sure customer doesn't have open order
			decoder := json.NewDecoder(req.Body)
			defer req.Body.Close()
			var order orderTrans
			err := decoder.Decode(&order)
			if err != nil {
				httpError(err.Error())
				return
			}
			fmt.Printf("order: %+v\n", order)
			inserts := make([]*bson.Element, 0)
			if order.Cust != "" {
				oid, err := objectid.FromHex(order.Cust)
				if err != nil {
					httpError(err.Error())
					return
				}
				inserts = append(inserts, bson.EC.ObjectID("cust", oid))
			}
			if order.Store != "" {
				oid, err := objectid.FromHex(order.Store)
				if err != nil {
					httpError(err.Error())
					return
				}
				inserts = append(inserts, bson.EC.ObjectID("store", oid))
			}
			fmt.Printf("inserts: %+v\n", inserts)
			inserter := bson.NewDocument()
			for _, update := range inserts {
				inserter.Append(update)
			}
			fmt.Printf("inserter: %+v\n", inserter)
			result, err := ordersColl.InsertOne(context.Background(), inserter, nil)
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
		} else {
			orderID := strings.TrimPrefix(req.URL.Path, "/api/v1/order/")
			log.Printf("post param: %s", orderID)
			httpError("this is done")
		}

	case "GET": // list items for order, id in path
		orderID := strings.TrimPrefix(req.URL.Path, "/api/v1/order/")
		log.Printf("get param: %s", orderID)
		oid, err := objectid.FromHex(orderID)
		if err != nil {
			httpError(err.Error())
			return
		}
		// todo: validate order id?
		filter := bson.NewDocument(bson.EC.ObjectID("store", oid))
		cur, err := orderItemsColl.Find(context.Background(), filter)
		if err != nil {
			httpError(err.Error())
			return
		}
		defer cur.Close(context.Background())
		list := make([]orderItem, 0)
		for cur.Next(context.Background()) {
			var item orderItem
			err := cur.Decode(&item)
			if err != nil {
				httpError(err.Error())
				return
			}
			fmt.Printf("item: %+v\n", item)
			list = append(list, item)
		}
		res.Header().Set("Content-Type", "application/json")
		json.NewEncoder(res).Encode(list)

	case "PUT": // add item
		// todo: make sure everything is specified and valid
		decoder := json.NewDecoder(req.Body)
		defer req.Body.Close()
		var item orderItem
		err := decoder.Decode(&item)
		if err != nil {
			httpError(err.Error())
			return
		}
		fmt.Printf("item: %+v\n", item)
		inserts := make([]*bson.Element, 0)
		if item.Order != "" {
			oid, err := objectid.FromHex(item.Order)
			if err != nil {
				httpError(err.Error())
				return
			}
			inserts = append(inserts, bson.EC.ObjectID("order", oid))
		}
		if item.Item != "" {
			oid, err := objectid.FromHex(item.Item)
			if err != nil {
				httpError(err.Error())
				return
			}
			inserts = append(inserts, bson.EC.ObjectID("item", oid))
		}
		if item.Count != 0 {
			inserts = append(inserts, bson.EC.Int32("count", int32(item.Count)))
		}
		fmt.Printf("inserts: %+v\n", inserts)
		inserter := bson.NewDocument()
		for _, update := range inserts {
			inserter.Append(update)
		}
		fmt.Printf("inserter: %+v\n", inserter)
		result, err := orderItemsColl.InsertOne(context.Background(), inserter, nil)
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
		itemID := strings.TrimPrefix(req.URL.Path, "/api/v1/order/")
		log.Printf("patch param: %s", itemID)
		oid, err := objectid.FromHex(itemID)
		if err != nil {
			httpError(err.Error())
			return
		}
		updater := bson.NewDocument(bson.EC.ObjectID("_id", oid))
		decoder := json.NewDecoder(req.Body)
		defer req.Body.Close()
		var item orderItem
		err = decoder.Decode(&item)
		if err != nil {
			httpError(err.Error())
			return
		}
		fmt.Printf("updater: %+v\n", updater)
		updates := make([]*bson.Element, 0)
		if item.Order != "" {
			httpError("Item order may not be changed")
		}
		if item.Item != "" {
			httpError("Item id may not be changed")
		}
		if item.Count != 0 {
			updates = append(updates, bson.EC.Int32("count", int32(item.Count)))
		}
		fmt.Printf("updates: %+v\n", updates)
		subdoc := bson.NewDocument()
		for _, update := range updates {
			subdoc.Append(update)
		}
		setter := bson.NewDocument(bson.EC.SubDocument("$set", subdoc))
		fmt.Printf("setter: %+v\n", setter)
		result, err := orderItemsColl.UpdateOne(context.Background(), updater, setter, nil)
		if err != nil {
			httpError(err.Error())
			return
		}
		fmt.Printf("result: %+v\n", result)
		res.Header().Set("Content-Type", "application/json")
		json.NewEncoder(res).Encode(result)

	case "DELETE": // delete item, id in path
		itemID := strings.TrimPrefix(req.URL.Path, "/api/v1/order/")
		log.Printf("patch param: %s", itemID)
		oid, err := objectid.FromHex(itemID)
		if err != nil {
			httpError(err.Error())
			return
		}
		deleter := bson.NewDocument(bson.EC.ObjectID("_id", oid))
		result, err := orderItemsColl.DeleteOne(context.Background(), deleter, nil)
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

func setupOrderItems() {
	ordersColl = database.Collection("orders")
	orderItemsColl = database.Collection("order_items")

	http.HandleFunc("/api/v1/order", handleOrderItems)
	http.HandleFunc("/api/v1/order/", handleOrderItems)
}
