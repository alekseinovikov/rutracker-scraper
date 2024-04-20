package main

import (
	"encoding/csv"
	"fmt"
	"github.com/gocolly/colly/v2"
	"log"
	"os"
	"sort"
	"strings"
)

const (
	domain      = "rutracker.org"
	siteAddress = "https://" + domain + "/forum"
	resultFile  = "games.csv"
)

var visitTopics = []string{
	"/viewforum.php?f=635",
	"/viewforum.php?f=127",
	"/viewforum.php?f=2203",
	"/viewforum.php?f=647",
	"/viewforum.php?f=646",
	"/viewforum.php?f=50",
	"/viewforum.php?f=53",
	"/viewforum.php?f=52",
	"/viewforum.php?f=54",
	"/viewforum.php?f=2226",
}

type GameData struct {
	Name       string
	Link       string
	SeedsCount int
}

func main() {
	var games []GameData

	collector := colly.NewCollector(
		colly.AllowedDomains(domain),
	)

	collector.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	collector.OnHTML(".hl-tr", func(topic *colly.HTMLElement) {
		var seedsCount = -1

		topic.ForEach(".vf-col-tor.tCenter.med.nowrap", func(i int, stats *colly.HTMLElement) {
			seedsCount = parseSeedsCount(stats)
		})

		topic.ForEach("a[href].tt-text", func(i int, link *colly.HTMLElement) {
			parseLinkDetails(seedsCount, link, &games)
		})
	})

	collector.OnHTML(".pg", func(pagination *colly.HTMLElement) {
		if strings.HasPrefix(pagination.Text, "След.") {
			link := pagination.Attr("href")
			pagination.Request.Visit(siteAddress + "/" + link)
		}
	})

	for _, topic := range visitTopics {
		err := collector.Visit(siteAddress + topic)
		if err != nil {
			log.Fatal(err)
		}

		collector.Wait()
	}

	sort.SliceStable(games, func(i, j int) bool {
		return games[i].SeedsCount > games[j].SeedsCount
	})

	writeToCsv(games)
}

func writeToCsv(games []GameData) {
	// write to csv
	file, err := os.OpenFile(resultFile, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	csvWriter := csv.NewWriter(file)
	resultStrings := make([][]string, 0, len(games)+1)
	resultStrings = append(resultStrings, []string{"Name", "Link", "Seeds"})

	for _, game := range games {
		resultStrings = append(resultStrings, []string{game.Name, game.Link, fmt.Sprint(game.SeedsCount)})
	}

	err = csvWriter.WriteAll(resultStrings)
	if err != nil {
		log.Fatal(err)
	}
	csvWriter.Flush()
}

func parseLinkDetails(seedsCount int, link *colly.HTMLElement, games *[]GameData) {
	if seedsCount == -1 {
		log.Println("seeds count is nil it's not a topic")
		return
	}

	linkText := link.Text
	absoluteLink := siteAddress + "/" + link.Attr("href")
	game := GameData{
		Name:       linkText,
		Link:       absoluteLink,
		SeedsCount: seedsCount,
	}

	*games = append(*games, game)
}

func parseSeedsCount(stats *colly.HTMLElement) int {
	statsText := stats.Text
	if !strings.Contains(statsText, "|") {
		return -1
	}
	seedsText := strings.Split(statsText, "|")[0]
	seedsText = strings.TrimSpace(seedsText)
	seedsText = strings.Trim(seedsText, "\n")
	seedsText = strings.Trim(seedsText, "\t")
	seedsText = strings.Trim(seedsText, "\r")
	seedsText = strings.Trim(seedsText, "\v")

	var seedsCount int
	_, err := fmt.Sscanf(seedsText, "%d", &seedsCount)
	if err != nil {
		log.Println("failed to parse seeds count", seedsText)
		return -1
	}

	return seedsCount
}
