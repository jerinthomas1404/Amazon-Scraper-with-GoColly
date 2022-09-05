package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type InnerDocument struct {
	Name         string `json:"name,omitempty" bson:"name,omitempty"`
	ImageURL     string `json:"imageURL,omitempty" bson:"imageURL,omitempty"`
	Desc         string `json:"description,omitempty" bson:"description,omitempty"`
	Price        string `json:"price,omitempty" bson:"price,omitempty"`
	TotalReviews int    `json:"totalReviews,omitempty" bson:"totalReviews,omitempty"`
}

type OuterDocument struct {
	ID         primitive.ObjectID `json:"_id,omitempty" bson:"_id,omitempty"`
	URL        string             `json:"url,omitempty" bson:"url,omitempty"`
	Product    InnerDocument      `json:"product,omitempty" bson:"product,omitempty"`
	LastUpdate time.Time          `json:"last_update, omitempty" bson:"last_update, omitempty"`
}

var c *mongo.Client

func checkData(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("content-type", "application/json")
	var new_doc, existing_doc OuterDocument
	_ = json.NewDecoder(r.Body).Decode(&new_doc)
	collection := c.Database("amazondb").Collection("amazoncollection")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	collection.FindOne(ctx, bson.M{"url": new_doc.URL}).Decode(&existing_doc)
	new_doc.LastUpdate = time.Now()
	if existing_doc.URL == "" {
		result, _ := collection.InsertOne(ctx, new_doc)
		json.NewEncoder(w).Encode(result)
	} else {
		result, _ := collection.UpdateOne(ctx,
			bson.M{"url": new_doc.URL},
			bson.D{
				primitive.E{
					Key: "$set",
					Value: bson.D{
						primitive.E{
							Key:   "product",
							Value: new_doc.Product,
						},
					},
				},
			},
		)
		json.NewEncoder(w).Encode(result)
	}
}

func getAllData(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Request Initiated")
	w.Header().Add("content-type", "application/json")
	var statuses []OuterDocument
	collection := c.Database("amazondb").Collection("amazoncollection")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "` + err.Error() + `"}`))
		return
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var status OuterDocument
		cursor.Decode(&status)
		statuses = append(statuses, status)
	}

	if err := cursor.Err(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message": "` + err.Error() + `"}`))
		return
	}

	json.NewEncoder(w).Encode(statuses)
}

func main() {
	localhost := "mongodb://database:27017"
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	clientOptions := options.Client().ApplyURI(localhost)
	c, _ = mongo.Connect(ctx, clientOptions)
	router := mux.NewRouter()
	router.HandleFunc("/aggregator", checkData).Methods("POST")
	router.HandleFunc("/aggregator", getAllData).Methods("GET")
	http.ListenAndServe(":8081", router)
}
