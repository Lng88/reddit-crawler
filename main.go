package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/gocolly/colly"
)

// TODO: message multiple users
var userID = "156579135878201346" // lng
// var userID = "239049979828764674"
var redditBaseUrl = "https://old.reddit.com"

func main() {
	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	config, err := LoadConfig(logger)
	if err != nil {
		logger.Fatal("Unable to load config:", err)
	}
	dg := NewDiscord(logger, config)
	ticker := time.NewTicker(time.Duration(config.ScrapeFrequency) * time.Minute)
	defer ticker.Stop()

	// Prepare for graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	// Ticker loop
	go func() {
		for {
			select {
			case <-ticker.C:
				logger.Println("Opening websocket")
				if err := dg.Open(); err != nil {
					logger.Println("Error opening connection:", err)
					continue
				}

				Scrape(logger, config.SubReddit, config.SearchStrings, dg)

				if err := dg.Close(); err != nil {
					logger.Println("Error closing connection:", err)
					continue
				}

				logger.Println("Websocket closed")
			case <-stopChan:
				logger.Println("Shutdown signal received, exiting...")
				return
			}
		}
	}()

	// Block main goroutine until an OS signal is received
	<-stopChan

	// Perform any cleanup and final operations here
	logger.Println("Application shutting down.")
}

func sendMessage(s *discordgo.Session, message string) error {
	ch, err := s.UserChannelCreate(userID)
	if err != nil {
		return fmt.Errorf("creating user channel: %w", err)
	}
	fmt.Println("channel ID", ch.ID)
	_, err = s.ChannelMessageSend(ch.ID, message)
	if err != nil {
		return fmt.Errorf("sending message: %w", err)
	}
	return nil
}

func Scrape(logger *log.Logger, subreddit string, substring []string, dg *discordgo.Session) {
	c := colly.NewCollector()

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		for _, v := range substring {
			pattern := fmt.Sprintf(`\[fs\].*?%s`, regexp.QuoteMeta(v))
			re, err := regexp.Compile(pattern)
			if err != nil {
				logger.Println("error compiling regexp", pattern)
				return
			}
			match := re.FindString(strings.ToLower(e.Text))
			if match != "" {
				err := sendMessage(dg, fmt.Sprintf("Found match for %s: %s/%s", v, redditBaseUrl, e.Attr("href")))
				if err != nil {
					logger.Fatal("error sending message: ", err)
				}
				fmt.Println("TEXT:", e.Text, "HREF:", e.Attr("href"))
			}
		}
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})

	c.Visit(fmt.Sprintf("%s/r/%s/new", redditBaseUrl, subreddit))
}

func NewDiscord(logger *log.Logger, config ConfigVars) *discordgo.Session {
	discord, err := discordgo.New("Bot " + config.Discord.BotToken)
	if err != nil {
		logger.Fatal("Unable to initialize Discord Bot")
	}
	return discord
}
