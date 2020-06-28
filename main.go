package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	// other imports
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
)

func main() {
	fmt.Println("short_memory bot v0.3.1")
	lambda.Start(handler)
}

type lambdaEvent struct{}

// Credentials stores all of our access/consumer tokens
// and secret keys needed for authentication against
// the twitter REST API.
type Credentials struct {
	ConsumerKey       string
	ConsumerSecret    string
	AccessToken       string
	AccessTokenSecret string
}

func handler(ctx context.Context, e lambdaEvent) error {
	creds := Credentials{
		AccessToken:       os.Getenv("TWITTER_ACCESS_TOKEN"),
		AccessTokenSecret: os.Getenv("TWITTER_ACCESS_TOKEN_SECRET"),
		ConsumerKey:       os.Getenv("TWITTER_API_KEY"),
		ConsumerSecret:    os.Getenv("TWITTER_API_SECRET"),
	}

	if creds.AccessToken == "" {
		return errors.New("Missing AccessToken")
	}
	if creds.AccessTokenSecret == "" {
		return errors.New("Missing AccessTokenSecret")
	}
	if creds.ConsumerKey == "" {
		return errors.New("Missing ConsumerKey")
	}
	if creds.ConsumerSecret == "" {
		return errors.New("Missing ConsumerSecret")
	}

	// Create authorized user client
	client, err := getUserClient(&creds)
	if err != nil {
		log.Println("Error getUserClient")
		log.Print(err)
		return err
	}

	maxAge := time.Now().AddDate(0, -1, 0) // 1 month ago
	// Find first tweet more than a month old
	tweet, err := getFirstTweetOlderThan(client, maxAge, 0)
	if err != nil {
		log.Println("Error getFirstTweetOlderThan")
		log.Print(err)
		return err
	}

	if tweet != nil {
		// Delete all tweets more than a month old
		err = deleteThisTweetAndOlder(client, tweet)
		if err != nil {
			log.Println("Error deleteThisTweetAndOlder")
			log.Print(err)
			return err
		}

		log.Print("Done")
	} else {
		log.Print("No tweets to delete")
	}

	return nil
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

func getFirstTweetOlderThan(client *twitter.Client, maxAge time.Time, maxID int64) (*twitter.Tweet, error) {
	f, t := false, true // todo: wtf fix this
	tweets, _, err := client.Timelines.UserTimeline(&twitter.UserTimelineParams{
		ExcludeReplies:  &f,
		IncludeRetweets: &t,
		TrimUser:        &t,
		Count:           100,
		MaxID:           maxID,
	})
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(tweets); i++ {
		createdAt, err := tweets[i].CreatedAtTime()
		if err != nil {
			return nil, err
		}

		if createdAt.Before(maxAge) {
			return &tweets[i], nil
		}
	}

	// only found MaxID tweet
	if len(tweets) == 1 {
		return nil, nil
	}

	return getFirstTweetOlderThan(client, maxAge, tweets[len(tweets)-1].ID)
}

func deleteThisTweetAndOlder(client *twitter.Client, tweet *twitter.Tweet) error {
	f, t := false, true // todo: wtf fix this
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
		return deleteThisTweetAndOlder(client, &tweets[len(tweets)-1])
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
	log.Printf("DELETED (%v): \"%v\"\n", destroyed.CreatedAt, destroyed.Text)
	return nil
}
