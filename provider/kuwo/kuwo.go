package kuwo

import (
	"encoding/base64"
	"encoding/json"
	"github.com/xiangism/UnblockNeteaseMusic/common"
	"github.com/xiangism/UnblockNeteaseMusic/network"
	"github.com/xiangism/UnblockNeteaseMusic/utils"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
)

func SearchSong(song common.SearchSong) (songs []*common.Song) {
	song.Keyword = strings.ToUpper(song.Keyword)
	song.Name = strings.ToUpper(song.Name)
	song.ArtistsName = strings.ToUpper(song.ArtistsName)
	token := getToken(song.Keyword)
	header := make(http.Header, 4)
	header["referer"] = append(header["referer"], "http://www.kuwo.cn/search/list?key="+url.QueryEscape(song.Keyword))
	header["csrf"] = append(header["csrf"], token)
	header["cookie"] = append(header["cookie"], "kw_token="+token)
	clientRequest := network.ClientRequest{
		Method:    http.MethodGet,
		RemoteUrl: "http://www.kuwo.cn/api/www/search/searchMusicBykeyWord?key=" + song.Keyword + "&pn=1&rn=30",
		//Host:      "www.kuwo.cn",
		Header: header,
		Proxy:  true,
	}
	resp, err := network.Request(&clientRequest)
	if err != nil {
		log.Println(err)
		return songs
	}

	if resp == nil || resp.Body == nil {
		return songs
	}

	defer resp.Body.Close()
	body, err := network.StealResponseBody(resp)
	if err != nil {
		log.Println(err)
		return songs
	}
	result := utils.ParseJsonV2(body)
	//log.Println(utils.ToJson(result))
	data, ok := result["data"].(common.MapType)
	if ok {
		list, ok := data["list"].([]interface{})
		if ok && len(list) > 0 {
			listLength := len(list)
			for index, matched := range list {
				if index >= listLength/2+1 || index > 9 {
					break
				}
				kowoSong, ok := matched.(common.MapType)

				if ok {
					rid, ok := kowoSong["rid"].(json.Number)
					if ok {
						//log.Println(utils.ToJson(kowoSong))
						songResult := &common.Song{}
						singerName, singerNameOk := kowoSong["artist"].(string)
						songName, songNameOk := kowoSong["name"].(string)
						//musicSlice := strings.Split(musicrid, "_")
						//musicId := musicSlice[len(musicSlice)-1]
						songResult.PlatformUniqueKey = kowoSong
						songResult.PlatformUniqueKey["UnKeyWord"] = song.Keyword
						songResult.Source = "kuwo"
						songResult.PlatformUniqueKey["header"] = header
						songResult.PlatformUniqueKey["musicId"] = rid.String()
						//songResult.PlatformUniqueKey["unblockId"] = strconv.FormatInt(rid, 10)
						songResult.Id = rid.String()
						if len(songResult.Id) > 0 {
							songResult.Id = string(common.KuWoTag) + songResult.Id
						}
						songResult.Name = songName
						songResult.Artist = singerName
						songResult.AlbumName, _ = kowoSong["album"].(string)
						songResult.Artist = strings.ReplaceAll(singerName, " ", "")
						if song.OrderBy == common.MatchedScoreDesc {
							if strings.Contains(songName, "伴奏") && !strings.Contains(song.Keyword, "伴奏") {
								continue
							}
							var songNameSores float32 = 0.0
							if songNameOk {
								//songNameKeys := utils.ParseSongNameKeyWord(songName)
								////log.Println("songNameKeys:", strings.Join(songNameKeys, "、"))
								//songNameSores = utils.CalMatchScores(searchSongName, songNameKeys)
								//log.Println("songNameSores:", songNameSores)
								songNameSores = utils.CalMatchScoresV2(song.Name, songName, "songName")
							}
							var artistsNameSores float32 = 0.0
							if singerNameOk {
								singerName = strings.ReplaceAll(singerName, "&", "、")
								//artistKeys := utils.ParseSingerKeyWord(singerName)
								////log.Println("kuwo:artistKeys:", strings.Join(artistKeys, "、"))
								//artistsNameSores = utils.CalMatchScores(searchArtistsName, artistKeys)
								artistsNameSores = utils.CalMatchScoresV2(song.ArtistsName, singerName, "singerName")
								//log.Println("kuwo:artistsNameSores:", artistsNameSores)
							}
							songMatchScore := songNameSores*0.6 + artistsNameSores*0.4
							songResult.MatchScore = songMatchScore
						} else if song.OrderBy == common.PlatformDefault {

						}
						songs = append(songs, songResult)
						//log.Println(utils.ToJson(searchSong))

					}
				}
			}

		}
	}
	if song.OrderBy == common.MatchedScoreDesc && len(songs) > 1 {
		sort.Sort(common.SongSlice(songs))
	}
	if song.Limit > 0 && len(songs) > song.Limit {
		songs = songs[:song.Limit]
	}
	return songs
}
func GetSongUrl(searchSong common.SearchMusic, song *common.Song) *common.Song {
	if id, ok := song.PlatformUniqueKey["musicId"]; ok {
		if musicId, ok := id.(string); ok {
			if httpHeader, ok := song.PlatformUniqueKey["header"]; ok {
				if header, ok := httpHeader.(http.Header); ok {
					//clientRequest := network.ClientRequest{
					//	Method:    http.MethodGet,
					//	RemoteUrl: "http://antiserver.kuwo.cn/anti.s?type=convert_url&format=mp3&response=url&rid=MUSIC_" + musicId,
					//	Host:      "antiserver.kuwo.cn",
					//	Header:    header,
					//	Proxy:     false,
					//}
					//header := make(http.Header, 1)
					header["user-agent"] = append(header["user-agent"], "okhttp/3.10.0")
					format := "flac|mp3"
					br := ""
					switch searchSong.Quality {
					case common.Standard:
						format = "mp3"
						br = "&br=128kmp3"
					case common.Higher:
						format = "mp3"
						br = "&br=192kmp3"
					case common.ExHigh:
						format = "mp3"
					case common.Lossless:
						format = "flac|mp3"
					default:
						format = "flac|mp3"
					}

					clientRequest := network.ClientRequest{
						Method:               http.MethodGet,
						ForbiddenEncodeQuery: true,
						RemoteUrl:            "http://mobi.kuwo.cn/mobi.s?f=kuwo&q=" + base64.StdEncoding.EncodeToString(Encrypt([]byte("corp=kuwo&p2p=1&type=convert_url2&sig=0&format="+format+"&rid="+musicId+br))),
						//Host:                 "mobi.kuwo.cn",
						Header:               header,
						Proxy:                true,
					}
					//log.Println(clientRequest.RemoteUrl)
					resp, err := network.Request(&clientRequest)
					if err != nil {
						log.Println(err)
						return song
					}
					defer resp.Body.Close()
					body, err := network.GetResponseBody(resp, false)
					reg := regexp.MustCompile(`http[^\s$"]+`)
					address := string(body)
					//log.Println(address)
					params := reg.FindStringSubmatch(address)
					//log.Println(params)
					if len(params) > 0 {
						song.Url = params[0]
						return song
					}

				}
			}
		}
	}
	return song
}
func ParseSong(searchSong common.SearchSong) *common.Song {
	songs := SearchSong(searchSong)

	for _, item := range songs {
		song := GetSongUrl(common.SearchMusic{Quality: searchSong.Quality}, item)
		if len(song.Url) > 0 {
			return song
		}
	}

	e := &common.Song{}
	return e
}
func getToken(keyword string) string {
	var token = ""
	clientRequest := network.ClientRequest{
		Method:    http.MethodGet,
		RemoteUrl: "http://kuwo.cn/search/list?key=" + keyword,
		Host:      "kuwo.cn",
		Header:    nil,
		Proxy:     false,
	}
	resp, err := network.Request(&clientRequest)
	if err != nil {
		log.Println(err)
		return token
	}
	if resp == nil || resp.Body == nil {
		return ""
	}
	defer resp.Body.Close()
	cookies := resp.Header.Get("set-cookie")
	if strings.Contains(cookies, "kw_token") {
		cookies = utils.ReplaceAll(cookies, ";.*", "")
		splitSlice := strings.Split(cookies, "=")
		token = splitSlice[len(splitSlice)-1]
	}
	return token
}
