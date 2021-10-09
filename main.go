package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"io"
	"models/user"
	"net/http"
	"os/user"
	"sync"
	"time"
)

var client *mongo.Client

var (
	mutex sync.Mutex
)

func createHash(key string) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

func encrypt(data []byte, passphrase string) []byte {
	block, _ := aes.NewCipher([]byte(createHash(passphrase)))
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext
}

func CreateUserEndpoint(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("content-type", "application/json")
	var instaUser = user.User
	ciphertext := encrypt([]byte(instaUser.Password), "password")
	instaUser.Password = ciphertext
	_ = json.NewDecoder(request.Body).Decode(&instaUser)
	collection := client.Database("appointy").Collection("users")
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	result, _ := collection.InsertOne(ctx, instaUser)
	json.NewEncoder(response).Encode(result)
}

func GetUserEndpoint(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("content-type", "application/json")
	params := mux.Vars(request)
	id, _ := primitive.ObjectIDFromHex(params["id"])
	var instaUser = user.User
	collection := client.Database("appointy").Collection("users")
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	err := collection.FindOne(ctx, instaUser{ID: id}).Decode(&instaUser)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
	json.NewEncoder(response).Encode(instaUser)
}

func GetEveryUserEndpoint(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("content-type", "application/json")
	var people []user.User
	collection := client.Database("appointy").Collection("users")
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var instaUser = user.User
		cursor.Decode(&instaUser)
		people = append(people, instaUser)
	}
	if err := cursor.Err(); err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
	json.NewEncoder(response).Encode(people)
	findOptions := options.Find()
	var perPage int = 10
	findOptions.SetSkip(5)
	findOptions.SetLimit(perPage)
}

func CreatePostEndpoint(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("content-type", "application/json")
	var instaPost = user.Post
	_ = json.NewDecoder(request.Body).Decode(&instaPost)
	collection := client.Database("appointy").Collection("posts")
	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)
	result, _ := collection.InsertOne(ctx, instaPost)
	json.NewEncoder(response).Encode(result)
}

func GetPostEndpoint(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("content-type", "application/json")
	params := mux.Vars(request)
	id, _ := primitive.ObjectIDFromHex(params["id"])
	var instaPost = user.Posts
	instaPost.Timestamp = time.Now()
	collection := client.Database("appointy").Collection("posts")
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	err := collection.FindOne(ctx, instaPost{ID: id}).Decode(&instaPost)
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
	json.NewEncoder(response).Encode(instaPost)
}

func GetEveryPostEndpoint(response http.ResponseWriter, request *http.Request) {
	response.Header().Set("content-type", "application/json")
	var post []user.Post
	collection := client.Database("appointy").Collection("posts")
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
	defer cursor.Close(ctx)
	for cursor.Next(ctx) {
		var instaPost = user.Post
		cursor.Decode(&instaPost)
		post = append(post, instaPost)
	}
	if err := cursor.Err(); err != nil {
		response.WriteHeader(http.StatusInternalServerError)
		response.Write([]byte(`{ "message": "` + err.Error() + `" }`))
		return
	}
	json.NewEncoder(response).Encode(post)
	findOptions := options.Find()
	var perPage int = 10
	findOptions.SetSkip(5)
	findOptions.SetLimit(perPage)
}

func UserFunctions(wg *sync.WaitGroup) {
	mutex.Lock()
	router := mux.NewRouter()
	router.HandleFunc("/Users", CreateUserEndpoint).Methods("POST")
	router.HandleFunc("/Users", GetEveryUserEndpoint).Methods("GET")
	router.HandleFunc("/Users/{id}", GetUserEndpoint).Methods("GET")
	http.ListenAndServe(":12345", router)
	mutex.Unlock()
	wg.Done()
}

func PostFunctions(wg *sync.WaitGroup) {
	mutex.Lock()
	router := mux.NewRouter()
	router.HandleFunc("/Posts", CreatePostEndpoint).Methods("POST")
	router.HandleFunc("/Posts", GetEveryPostEndpoint).Methods("GET")
	router.HandleFunc("/Posts/{id}", GetPostEndpoint).Methods("GET")
	http.ListenAndServe(":12346", router)
	mutex.Unlock()
	wg.Done()
}

func main() {
	fmt.Println("Welcome to Appointy Internship...")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	clientOptions := options.Client().ApplyURI("mongodb://localhost:27017")
	client, _ = mongo.Connect(ctx, clientOptions)
	var wg sync.WaitGroup
	UserFunctions(&wg)
	PostFunctions(&wg)
}
