package main

import (
	"fmt"
	"log"
	"os"

	// other imports
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("short_memory bot v0.1.0")
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	creds := Credentials{
		AccessToken:       os.Getenv("TWITTER_ACCESS_TOKEN"),
		AccessTokenSecret: os.Getenv("TWITTER_ACCESS_TOKEN_SECRET"),
		ConsumerKey:       os.Getenv("TWITTER_API_KEY"),
		ConsumerSecret:    os.Getenv("TWITTER_API_SECRET"),
	}

	fmt.Printf("%+v\n", creds)

	// Login Twitter Client
	client, err := getUserClient(&creds)
	if err != nil {
		log.Println("Error getUserClient")
		log.Fatal(err)
	}

	// Find 100th tweet
	tweet, err := getHundredthTweet(client)
	if err != nil {
		log.Println("Error getHundredthTweet")
		log.Fatal(err)
	}

	if tweet != nil {
		// delete older tweets
		err = deleteEverythingOlderThanTweet(client, tweet)
		if err != nil {
			log.Println("Error deleteEverythingOlderThanTweet")
			log.Fatal(err)
		}
	} else {
		log.Print("No tweets to delete!")
	}
}

// Credentials stores all of our access/consumer tokens
// and secret keys needed for authentication against
// the twitter REST API.
type Credentials struct {
	ConsumerKey       string
	ConsumerSecret    string
	AccessToken       string
	AccessTokenSecret string
}

func getUserClient(creds *Credentials) (*twitter.Client, error) {
	config := oauth1.NewConfig(creds.ConsumerKey, creds.ConsumerSecret)
	token := oauth1.NewToken(creds.AccessToken, creds.AccessTokenSecret)

	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)

	verifyParams := &twitter.AccountVerifyParams{
		SkipStatus:   twitter.Bool(true),
		IncludeEmail: twitter.Bool(false),
	}

	user, _, err := client.Accounts.VerifyCredentials(verifyParams)
	if err != nil {
		return nil, err
	}

	log.Printf("Logged in as: %s", user.ScreenName)

	return client, nil
}

func tweet(client *twitter.Client, text string) (*twitter.Tweet, error) {
	tweet, _, err := client.Statuses.Update(text, nil)
	if err != nil {
		return nil, err
	}
	log.Printf("%+v\n", tweet)
	return tweet, nil
}

func getHundredthTweet(client *twitter.Client) (*twitter.Tweet, error) {
	f, t := false, true // wtf todo: fix this
	tweets, _, err := client.Timelines.UserTimeline(&twitter.UserTimelineParams{
		ExcludeReplies:  &f,
		IncludeRetweets: &t,
		TrimUser:        &t,
		Count:           100,
	})

	if err != nil {
		return nil, err
	}

	if len(tweets) < 100 {
		return nil, nil
	}

	return &tweets[len(tweets)-1], nil
}

func deleteEverythingOlderThanTweet(client *twitter.Client, tweet *twitter.Tweet) error {
	f, t := false, true // wtf todo: fix this
	tweets, _, err := client.Timelines.UserTimeline(&twitter.UserTimelineParams{
		ExcludeReplies:  &f,
		IncludeRetweets: &t,
		TrimUser:        &t,
		Count:           100,
		MaxID:           tweet.ID,
	})

	if err != nil {
		return err
	}
	if len(tweets) == 0 {
		return nil
	}

	err = deleteTweets(client, tweets)

	if len(tweets) == 100 {
		return deleteEverythingOlderThanTweet(client, &tweets[len(tweets)-1])
	}
	return nil
}

func deleteTweets(client *twitter.Client, tweets []twitter.Tweet) error {
	for i := 0; i < len(tweets); i++ {
		err := deleteTweet(client, tweets[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func deleteTweet(client *twitter.Client, tweet twitter.Tweet) error {
	destroyed, _, err := client.Statuses.Destroy(tweet.ID, nil)
	if err != nil {
		return err
	}
	log.Printf("DELETED TWEET: %+v\n", destroyed)
	return nil
}
