package notifier

import (
	"errors"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Notifier struct {
	client         http.Client
	latestPostDate time.Time
}

type post struct {
	attachments [3]string
	comment     string
	subject     string
	date        time.Time
	author      string
}

const platformURL = "https://www.platforma.mechaniktg.pl/"

// Create creates lurker instance
func Create() *Notifier {
	j, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}
	return &Notifier{http.Client{
		Transport:     nil,
		CheckRedirect: nil,
		Jar:           j,
		Timeout:       0,
	},
		time.Now(),
		// time.Date(2020, time.October, 24, 10, 30, 52, 0, time.UTC), // time for testing purposes
	}
}

// Login performs a log in to a platform and pesrsits it via session cookie
func (l *Notifier) Login(login, password string) error {
	resp, err := l.client.PostForm(platformURL, url.Values{
		"login": {login},
		"haslo": {password},
		"s2":    {"Zaloguj+jako+uczeń"},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("status code error: %d %s", resp.StatusCode, resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	// Check if succesfully logged
	if doc.Find("#artykul span.tekststandard").First().Text() == "Nie ma takiego konta ucznia. Upewnij się czy wpisane dane są poprawne." {
		return errors.New("Could not log in")
	} else {
		return nil
	}
}

func (l *Notifier) FetchPosts() ([]post, error) {
	resp, err := l.client.PostForm(platformURL, url.Values{
		"u1": {"Przeglądaj+posty"},
	})
	if err != nil {
		log.Println(err)
		return nil, errors.New("Could not fetch posts")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Println("status code error: %d %s", resp.StatusCode, resp.Status)
		return nil, errors.New("Could not fetch posts")
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		log.Println(err)
		return nil, errors.New("Could not parse post site")
	}
	retPosts := []post{}
	posts := doc.Find("#postynauczycieli")
	posts.Each(func(i int, s *goquery.Selection) {
		p := post{}
		// Parse attachments
		s.Find("#postynauczycieli tbody td.w150 a").EachWithBreak(func(i int, a *goquery.Selection) bool {
			if a.Length() == 0 {
				return false
			}
			p.attachments[i], _ = a.Attr("href")
			return true
		})
		// Parse comment
		p.comment = s.Find(".w300").Text()
		// Parse subject
		p.subject = s.Find(".w150:nth-child(5)").Text()
		// Parse date and author (complicated regexp fun to just split date and autor :P)
		re := regexp.MustCompile(`\d{4}-\d{1,2}-\d{1,2} \d+:\d+:\d+`)
		str := s.Find(".w150:nth-child(7)").Text()
		loc := re.FindIndex([]byte(str))
		date := string(str[loc[0]:loc[1]])
		parsedTime, err := time.Parse("2006-01-2 15:04:05", date)
		if err != nil {
			log.Print("Problem parsing time:", err)
		}
		p.date = parsedTime

		author := string(str[loc[1]:])
		p.author = author

		// println(p.attachments[0])
		// println(p.attachments[1])
		// println(p.comment)
		// println(p.date.String())
		// println(p.author)
		// println(p.subject)
		// println("---")
		retPosts = append(retPosts, p)
	})

	return retPosts, nil
}

// helper function to reverse order of posts
func reversePostArray(n []post) []post {
	l := len(n)
	rev := make([]post, l)
	copy(rev, n)
	for i, j := 0, l-1; i < j; i, j = i+1, j-1 {
		rev[i] = n[j]
		rev[j] = n[i]
	}
	return rev
}

// NotifyAboutLatestPosts fetches last posts and notifyies new posts
func (l *Notifier) NotifyAboutLatestPosts(gotifyURL string) {
	p, err := l.FetchPosts()
	if err != nil {
		log.Println("Error when fetching posts")
		return
	}
	posts := reversePostArray(p)
	for _, post := range posts {
		// Check if post is newer than the latest post
		if post.date.After(l.latestPostDate) {
			resp, err := l.client.PostForm(gotifyURL,
				url.Values{"message": {post.comment}, "title": {post.author}, "priority": {"5"}})
			if err != nil {
				log.Fatalln("Error when notifycaying: ", err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				log.Println("Successfully notified about post from ", post.author)
			}
			// Set this post as latest after notified
			l.latestPostDate = post.date
		} else {
			return
		}
	}
	return
}
