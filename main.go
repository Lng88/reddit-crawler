package main

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

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
		logger.Fatal("unable to load config...")
	}
	dg := NewDiscord(logger, config)
	err = dg.Open()
	if err != nil {
		logger.Fatal("error opening connection,", err)
		return
	}

	// TODO: Make cron job

	Scrape(logger, config.SubReddit, config.SearchStrings, dg)
	logger.Println("closing websocket")
	dg.Close()
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
