package main

import (
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/sclevine/agouti"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

func main() {

	//getTwitterId() // フォワーの多い人のIDを取得
	go GetToken()
	go GetToken()
	go GetToken()
	go GetToken()
	time.Sleep(time.Second * 10)
	getListName()
	//Get()
}

func getTwitterId() {
	db := GetDb()
	//
	for i := 12850; i < 79250; i += 50 {
		fmt.Println(i, 79250-i)
		u := "https://meyou.jp/ranking/follower_allcat/" + strconv.Itoa(i)
		doc, err := goquery.NewDocument(u)
		if err != nil {
			panic(err)
		}
		doc.Find("span.author-username").Each(func(i int, selection *goquery.Selection) {
			id := strings.TrimSpace(strings.Trim(selection.Text(), "@"))
			var info Info
			db.Find(&info, "name = ?", id)
			if info.ID != 0 {
				return
			}
			info.Name = id
			db.Create(&info)
		})
	}
}

const BASE_URL = "https://api.twitter.com/1.1/lists/ownerships.json?include_profile_interstitial_type=1&include_blocking=1&include_blocked_by=1&include_followed_by=1&include_want_retweets=1&include_mute_edge=1&include_can_dm=1&include_can_media_tag=1&skip_status=1&cards_platform=Web-12&include_cards=1&include_composer_source=true&include_ext_alt_text=true&include_reply_count=1&tweet_mode=extended&cursor=-1&"

func GetToken(){
	for {
		var err error
		driver := agouti.ChromeDriver(agouti.ChromeOptions("args", []string{"--headless"}))
		err = driver.Start()
		if err != nil{
			panic(err)
		}
		page,err := driver.NewPage()
		if err != nil{
			panic(err)
		}
		err = page.Navigate("https://twitter.com/1919/lists")
		if err != nil{
			panic(err)
		}
		c,err := page.GetCookies()
		if err !=nil {
			panic(err)
		}
		for _,cs := range c{
			if cs.Name == "gt"{
				keys = append(keys,cs.Value)
			}
		}
		_ = driver.Stop()
		fmt.Println(len(keys),now,keys[now])
	}
}

var keys []string
var now int
var old int

func getListName() {
	db := GetDb()
	var infos []Info
	db.Find(&infos)

	wg := sync.WaitGroup{}
	ch := make(chan int,5)
	now = 0
	old = 0

	for _, info := range infos {
		if info.All == `{"errors":[{"message":"Rate limit exceeded","code":88}]}` {
			continue
		}
		if info.All == `{"errors":[{"code":215,"message":"Bad Authentication data."}]}` {
			continue
		}
		if info.All == `{"errors":[{"code":34,"message":"Sorry, that page does not exist."}]}`{
			continue
		}
		if info.List != "" {
			continue
		}
		wg.Add(1)
		ch<-1
		fmt.Print(".")
		go func(db *gorm.DB,info Info) {
			defer wg.Done()
			defer func() {<-ch}()

			re:

			u := BASE_URL + "screen_name=" + info.Name
			req, err := http.NewRequest(http.MethodGet, u, nil)
			if err != nil {
				panic(err)
			}
			req.Header.Add("authorization", "Bearer AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA")
			req.Header.Add("x-guest-token",keys[now])

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				panic(err)
			}
			b, _ := ioutil.ReadAll(resp.Body)
			info.All = string(b)
			if resp.StatusCode != http.StatusOK {
				if string(b) == `{"errors":[{"code":34,"message":"Sorry, that page does not exist."}]}`{
					info.All = string(b)
					db.Save(&info)
					return
				}
				fmt.Println(string(b),info.Name)
				if old == now{
					now++
					time.Sleep(time.Second * 5)
					old++
					log.Println(now)
				}
				goto re
			}

			if string(b) != `{"next_cursor":0,"next_cursor_str":"0","previous_cursor":0,"previous_cursor_str":"0","lists":[]}` {
				var twJson TwJson
				err = json.Unmarshal(b, &twJson)
				if err != nil {
					panic(err)
				}
				for _, l := range twJson.Lists {
					info.List += l.Name + SPLIT_KEY
					info.Uri += l.URI + SPLIT_KEY
				}
			}
			if info.List == "" {
				info.List = "Not Found"
			}
			db.Save(&info)
		}(db,info)
	}
	wg.Wait()
}

func GetDb() *gorm.DB {
	db, err := gorm.Open("sqlite3", "./main.db")
	if err != nil {
		panic(err)
	}

	// スキーマのマイグレーション
	db.AutoMigrate(&Info{})
	return db
}

const SPLIT_KEY = "^~^"

type Info struct {
	gorm.Model
	Name string `gorm:"unique"`
	List string
	Uri  string
	All  string
}

type TwJson struct {
	NextCursor        int    `json:"next_cursor"`
	NextCursorStr     string `json:"next_cursor_str"`
	PreviousCursor    int    `json:"previous_cursor"`
	PreviousCursorStr string `json:"previous_cursor_str"`
	Lists             []struct {
		ID              int    `json:"id"`
		IDStr           string `json:"id_str"`
		Name            string `json:"name"`
		URI             string `json:"uri"`
		SubscriberCount int    `json:"subscriber_count"`
		MemberCount     int    `json:"member_count"`
		Mode            string `json:"mode"`
		Description     string `json:"description"`
		Slug            string `json:"slug"`
		FullName        string `json:"full_name"`
		CreatedAt       string `json:"created_at"`
		Following       bool   `json:"following"`
		User            struct {
			ID          int    `json:"id"`
			IDStr       string `json:"id_str"`
			Name        string `json:"name"`
			ScreenName  string `json:"screen_name"`
			Location    string `json:"location"`
			Description string `json:"description"`
			URL         string `json:"url"`
			Entities    struct {
				URL struct {
					Urls []struct {
						URL         string `json:"url"`
						ExpandedURL string `json:"expanded_url"`
						DisplayURL  string `json:"display_url"`
						Indices     []int  `json:"indices"`
					} `json:"urls"`
				} `json:"url"`
				Description struct {
					Urls []interface{} `json:"urls"`
				} `json:"description"`
			} `json:"entities"`
			Protected                      bool          `json:"protected"`
			FollowersCount                 int           `json:"followers_count"`
			FastFollowersCount             int           `json:"fast_followers_count"`
			NormalFollowersCount           int           `json:"normal_followers_count"`
			FriendsCount                   int           `json:"friends_count"`
			ListedCount                    int           `json:"listed_count"`
			CreatedAt                      string        `json:"created_at"`
			FavouritesCount                int           `json:"favourites_count"`
			UtcOffset                      interface{}   `json:"utc_offset"`
			TimeZone                       interface{}   `json:"time_zone"`
			GeoEnabled                     bool          `json:"geo_enabled"`
			Verified                       bool          `json:"verified"`
			StatusesCount                  int           `json:"statuses_count"`
			MediaCount                     int           `json:"media_count"`
			Lang                           interface{}   `json:"lang"`
			ContributorsEnabled            bool          `json:"contributors_enabled"`
			IsTranslator                   bool          `json:"is_translator"`
			IsTranslationEnabled           bool          `json:"is_translation_enabled"`
			ProfileBackgroundColor         string        `json:"profile_background_color"`
			ProfileBackgroundImageURL      string        `json:"profile_background_image_url"`
			ProfileBackgroundImageURLHTTPS string        `json:"profile_background_image_url_https"`
			ProfileBackgroundTile          bool          `json:"profile_background_tile"`
			ProfileImageURL                string        `json:"profile_image_url"`
			ProfileImageURLHTTPS           string        `json:"profile_image_url_https"`
			ProfileBannerURL               string        `json:"profile_banner_url"`
			ProfileImageExtensionsAltText  interface{}   `json:"profile_image_extensions_alt_text"`
			ProfileBannerExtensionsAltText interface{}   `json:"profile_banner_extensions_alt_text"`
			ProfileLinkColor               string        `json:"profile_link_color"`
			ProfileSidebarBorderColor      string        `json:"profile_sidebar_border_color"`
			ProfileSidebarFillColor        string        `json:"profile_sidebar_fill_color"`
			ProfileTextColor               string        `json:"profile_text_color"`
			ProfileUseBackgroundImage      bool          `json:"profile_use_background_image"`
			HasExtendedProfile             bool          `json:"has_extended_profile"`
			DefaultProfile                 bool          `json:"default_profile"`
			DefaultProfileImage            bool          `json:"default_profile_image"`
			PinnedTweetIds                 []interface{} `json:"pinned_tweet_ids"`
			PinnedTweetIdsStr              []interface{} `json:"pinned_tweet_ids_str"`
			HasCustomTimelines             bool          `json:"has_custom_timelines"`
			CanDm                          interface{}   `json:"can_dm"`
			CanMediaTag                    interface{}   `json:"can_media_tag"`
			Following                      interface{}   `json:"following"`
			FollowRequestSent              interface{}   `json:"follow_request_sent"`
			Notifications                  interface{}   `json:"notifications"`
			Muting                         interface{}   `json:"muting"`
			Blocking                       interface{}   `json:"blocking"`
			BlockedBy                      interface{}   `json:"blocked_by"`
			WantRetweets                   interface{}   `json:"want_retweets"`
			AdvertiserAccountType          string        `json:"advertiser_account_type"`
			AdvertiserAccountServiceLevels []interface{} `json:"advertiser_account_service_levels"`
			ProfileInterstitialType        string        `json:"profile_interstitial_type"`
			BusinessProfileState           string        `json:"business_profile_state"`
			TranslatorType                 string        `json:"translator_type"`
			FollowedBy                     interface{}   `json:"followed_by"`
			RequireSomeConsent             bool          `json:"require_some_consent"`
		} `json:"user"`
	} `json:"lists"`
}
