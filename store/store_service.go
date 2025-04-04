package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
	_ "github.com/lib/pq"
)

type StorageService struct {
	redisClient *redis.Client
	dbClient    *sql.DB
}

var (
	storeService = &StorageService{}
	ctx          = context.Background()
)

type URLMapping struct {
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
}

const cacheDuration = 6 * time.Hour

func InitializeStore() *StorageService {
	// er := godotenv.Load()
	// if er != nil {
	// 	log.Println("Warning: No .env file found, using default values")
	// 	panic(er)
	// }
	tempDB, errr := strconv.Atoi(os.Getenv("DB"))
	if errr != nil {
		log.Println("Warning: Invalid REDIS_DB value, defaulting to 0")
		tempDB = 0
	}
	redisClient := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("Password"),
		DB:       tempDB,
	})
	fmt.Println(redisClient.Options().Addr)
	pong, err := redisClient.Ping(ctx).Result()
	if err != nil {
		panic(fmt.Sprintf("Error init Redis: %v", err))
	}
	fmt.Printf("\n Redis started successfully: pong message = {%s} ", pong)
	storeService.redisClient = redisClient
	return storeService
}

func InitializeDb() *sql.DB {
	// err := godotenv.Load()
	// if err != nil {
	// 	log.Println("Warning: No .env file found, using default values")
	// 	panic(err)
	// }

	// host := os.Getenv("host")
	// portStr := os.Getenv("port")
	// user := os.Getenv("user")
	// password := os.Getenv("password")
	// dbname := os.Getenv("dbname")

	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	dbname := os.Getenv("POSTGRES_DB")

	// port, err := strconv.Atoi(portStr)
	// if err != nil {
	// 	log.Println("Warning: Invalid DB_PORT value, defaulting to 5432")
	// 	port = 5433
	// }
	fmt.Println(port)
	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s "+
		"password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	fmt.Println(db.Stats())
	err = db.Ping()
	if err != nil {
		db.Close()
		panic(err)
	}
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS urls (
		id SERIAL PRIMARY KEY,
		short_url VARCHAR(255) UNIQUE NOT NULL,
		long_url TEXT NOT NULL
	);`
	_, err = db.Exec(createTableQuery)
	if err != nil {
		panic(fmt.Sprintf("Error creating table: %v", err))
	}
	storeService.dbClient = db
	fmt.Println("Successfully connected!")
	return db
}

func CheckExistenceOfUrl(originalURL string) (bool, string) {
	query := `SELECT short_url FROM urls WHERE long_url=$1 LIMIT 1`
	var short string
	err := storeService.dbClient.QueryRow(query, originalURL).Scan(&short)
	if err == sql.ErrNoRows {
		return false, ""
	}
	if err != nil {
		log.Printf("Error checking URL existence: %v", err)
		return false, ""
	}
	return true, short
}

func SaveUrlMapping(shortURL string, originalURL string, userId string) {
	data := URLMapping{
		OriginalURL: originalURL,
		UserID:      userId,
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal URL mapping: %v", err))
	}
	err = storeService.redisClient.Set(ctx, shortURL, jsonData, cacheDuration).Err()
	if err != nil {
		panic(fmt.Sprintf("Failed saving key url | Error: %v - shortURL: %s - originalURL: %s\n", err, shortURL, originalURL))
	}

	query := `INSERT INTO urls (short_url, long_url) VALUES ($1, $2) RETURNING id`
	var id int
	err = storeService.dbClient.QueryRow(query, shortURL, originalURL).Scan(&id)
	if err != nil {
		log.Printf("Failed to insert URL mapping into database | Error: %v", err)
		return
	}

	fmt.Printf("URL successfully stored in PostgreSQL with ID: %d\n", id)
}

func RetriveInitialUrlFromPG(originalURL string) (bool, string) {
	query := `SELECT long_url FROM urls WHERE short_url=$1 LIMIT 1`
	var longURl string
	err := storeService.dbClient.QueryRow(query, originalURL).Scan(&longURl)
	if err == sql.ErrNoRows {
		return false, ""
	}
	if err != nil {
		log.Printf("Error checking URL existence: %v", err)
		return false, ""
	}
	return true, longURl
}

func RetriveInitialUrl(shortURL string) (string, string, error) {
	jsonData, err := storeService.redisClient.Get(ctx, shortURL).Result()
	if err == redis.Nil {
		exist, longURL := RetriveInitialUrlFromPG(shortURL)
		fmt.Println("Fetched from pg")
		if !exist {
			return "", "", fmt.Errorf("URL not found in cache and database: %s", shortURL)
		} else {
			data := URLMapping{OriginalURL: longURL, UserID: ""}
			jsonData, _ := json.Marshal(data)
			storeService.redisClient.Set(ctx, shortURL, jsonData, cacheDuration)
		}
		return longURL, "", nil
	} else if err != nil {
		return "", "", fmt.Errorf("failed retrieving url | Error: %v - shortURL: %s", err, shortURL)
	}
	var data URLMapping
	err = json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		return "", "", fmt.Errorf("failed unmarshaling url data | Error: %v - shortURL: %s", err, shortURL)
	}

	return data.OriginalURL, data.UserID, nil
}
