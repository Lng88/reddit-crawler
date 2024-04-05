package main

import (
	"bufio"
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

var redditBaseUrl = "https://old.reddit.com"

// must exist for program to work
var fileName = "seen.txt"
var fileMap = map[string]struct{}{}

func main() {
	logger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	config, err := LoadConfig(logger)
	if err != nil {
		logger.Fatal("Unable to load config:", err)
	}
	dg := NewDiscord(logger, config)
	ticker := time.NewTicker(time.Duration(config.ScrapeFrequency) * time.Minute)
	defer ticker.Stop()
	wipeFileTicker := time.NewTicker(24 * time.Hour)

	file, err := os.Create(fileName)
	if err != nil {
		log.Fatalf("Failed to create file: %s", err)
	}
	file.Close()

	// Prepare for graceful shutdown
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	logger.Println("waiting for next scrape...")
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

				Scrape(logger, config.SubReddit, config.SearchStrings, dg, config.UsersToMsg)

				if err := dg.Close(); err != nil {
					logger.Println("Error closing connection:", err)
					continue
				}

				logger.Println("Websocket closed")
			case <-wipeFileTicker.C:
				os.Truncate(fileName, 0)
				fileMap = map[string]struct{}{}
			case <-stopChan:
				logger.Println("Shutdown signal received, exiting...")
				return
			}
		}
	}()

	// Block main goroutine until an OS signal is received
	<-stopChan

	logger.Println("Application shutting down.")
}

func writeFile(content string) error {
	file, err := os.OpenFile(fileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.WriteString(content + "\n"); err != nil {
		return err
	}

	return nil
}

func readFileLineByLine() error {
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a new Scanner for the file
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fileMap[line] = struct{}{}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func sendMessage(logger *log.Logger, s *discordgo.Session, message string, users []string) error {
	for _, u := range users {
		ch, err := s.UserChannelCreate(u)
		if err != nil {
			return fmt.Errorf("creating user channel: %w", err)
		}
		fmt.Println("channel ID", ch.ID)
		_, err = s.ChannelMessageSend(ch.ID, message)
		if err != nil {
			logger.Println("warning: error sending message to: ", u, err)
		}
	}
	return nil
}

func Scrape(logger *log.Logger, subreddit string, substring []string, dg *discordgo.Session, users []string) {
	c := colly.NewCollector()

	c.OnHTML("a[href]", func(e *colly.HTMLElement) {
		for _, v := range substring {
			re, err := regexp.Compile(v)
			if err != nil {
				logger.Println("error compiling regexp", v)
				return
			}
			match := re.FindString(strings.ToLower(e.Text))
			if match != "" {
				err := readFileLineByLine()
				if err != nil {
					logger.Println("error reading file", err)
					return
				}
				if _, ok := fileMap[e.Text]; !ok {
					err = sendMessage(logger, dg, fmt.Sprintf("Found match for %s: %s/%s", v, redditBaseUrl, e.Attr("href")), users)
					if err != nil {
						logger.Println("ERROR: sending message: ", err)
						return
					}
					writeFile(e.Text)
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
	logger.Println("Initializing Discord agent...")
	discord, err := discordgo.New("Bot " + config.Discord.BotToken)
	if err != nil {
		logger.Fatal("Unable to initialize Discord Bot")
	}
	return discord
}
