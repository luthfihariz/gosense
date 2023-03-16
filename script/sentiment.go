package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	fmt.Println("Hello")
	RunSentiment()
}

var ctx = context.Background()

type news struct {
	Probability     float64 `bson:"probability"`
	Headline        string  `bson:"headline"`
	Url             string  `bson:"url"`
	ArticleBody     string  `bson:"articleBody"`
	ArticleBodyHtml string  `bson:"articleBodyHtml"`
	DatePublished   string  `bsob:"datePublished"`
}

type NewsSentiment struct {
	NewsHeadline    string
	Url             string
	ArticleBody     string
	ArticleBodyHtml string
	DatePublished   string
	PositiveScore   float64
	NegativeScore   float64
	NeutralScore    float64
}

type SentimentAnalysisApiResponse struct {
	Data [][]SentimentAnalysisResult
}

type SentimentAnalysisResult struct {
	Label string  `json:"label"`
	Score float64 `json:"score"`
}

type RequestError struct {
	StatusCode int
	Err        error
}

type ApiResponseError struct {
	Error string `json:"error"`
}

func (r *RequestError) Error() string {
	return fmt.Sprintf("status %d: err %v", r.StatusCode, r.Err)
}

func connect() (*mongo.Database, error) {
	clientOptions := options.Client()
	clientOptions.ApplyURI("mongodb://localhost:27017")
	client, err := mongo.NewClient(clientOptions)

	if err != nil {
		return nil, err
	}

	err = client.Connect(ctx)
	if err != nil {
		return nil, err
	}

	return client.Database("gosense"), nil
}

func getAll() []news {
	db, err := connect()
	if err != nil {
		log.Fatal(err.Error())
	}

	filter := bson.D{{Key: "probability", Value: bson.D{{Key: "$gte", Value: 0.99}}}}

	cursor, err := db.Collection("news").Find(ctx, filter)
	if err != nil {
		log.Fatal(err.Error())
	}

	defer cursor.Close(ctx)

	result := make([]news, 0)
	for cursor.Next(ctx) {
		var row news
		err := cursor.Decode(&row)
		if err != nil {
			log.Fatal(err.Error())
		}

		result = append(result, row)
	}

	return result
}

func bulkAddNewsSentiment(newsSentiments []NewsSentiment) int64 {
	db, err := connect()
	if err != nil {
		log.Fatal(err.Error())
	}

	collection := db.Collection("news_sentiment")
	writeModel := make([]mongo.WriteModel, 0)
	for _, news := range newsSentiments {
		insertModel := mongo.NewInsertOneModel().SetDocument(news)
		writeModel = append(writeModel, insertModel)
	}
	result, err := collection.BulkWrite(context.TODO(), writeModel)
	if err != nil {
		log.Fatal(err.Error())
	}

	fmt.Println(result)
	return result.InsertedCount
}

func RunSentiment() {
	rawNews := getAll()
	newsSentiments := make([]NewsSentiment, 0)
	for _, news := range rawNews {
		fmt.Println(news.Headline)
		apiResponse, err := getSentimentAnalysis(news)
		if err != nil {
			fmt.Println(fmt.Sprintf("Err %s", err.Error()))
			continue
		}

		fmt.Println(apiResponse)
		newsSentiment := NewsSentiment{
			Url:             news.Url,
			ArticleBody:     news.ArticleBody,
			NewsHeadline:    news.Headline,
			DatePublished:   news.DatePublished,
			ArticleBodyHtml: news.ArticleBodyHtml,
		}
		for _, res := range apiResponse {
			if res.Label == "LABEL_0" {
				newsSentiment.PositiveScore = res.Score
			} else if res.Label == "LABEL_1" {
				newsSentiment.NeutralScore = res.Score
			} else if res.Label == "LABEL_2" {
				newsSentiment.NegativeScore = res.Score
			}
		}

		newsSentiments = append(newsSentiments, newsSentiment)
	}

	insertedCount := bulkAddNewsSentiment(newsSentiments)
	fmt.Printf("Inserted %d row", insertedCount)
}

var hfInferenceAPI = "https://api-inference.huggingface.co/models/mdhugol/indonesia-bert-sentiment-classification"

func getSentimentAnalysis(item news) ([]SentimentAnalysisResult, error) {
	client := &http.Client{}
	data := SentimentAnalysisApiResponse{}
	textToBeAnalyze := fmt.Sprintf("%s %s", item.Headline, item.ArticleBody)

	jsonBody := []byte(fmt.Sprintf(`{"input":%s}`, textToBeAnalyze))
	request, err := http.NewRequest("POST", hfInferenceAPI, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Authorization", "Bearer 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")

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	if response.StatusCode != 200 {
		apiError := ApiResponseError{}
		return nil, &RequestError{response.StatusCode, json.NewDecoder(response.Body).Decode(&apiError)}
	}

	err = json.NewDecoder(response.Body).Decode(&data.Data)
	if err != nil {
		return nil, err
	}

	return data.Data[0], nil
}
