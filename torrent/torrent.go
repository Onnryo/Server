package torrent

import (
	"eve/errors"
	"net/http"
	"fmt"
	"time"
	//"bytes"
	"strings"
	"database/sql"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"io/ioutil"
	piratebay "github.com/gnur/go-piratebay"
	_"github.com/go-sql-driver/mysql"
	)

type Torrent struct {
	Name string `json:"name"`
	Id string `json:"id"`
	Uri string `json:"uri"`
	Title string `json:"title"`
	TorrentType string `json:"torrenttype"`
	ShowType string `json:"showtype"`
	Season string `json:"season"`
	Episode string `json:"episode"`
	User string `json: "user"`
	UserId int `json: "uid"`
	Query string `json: "query"`
}

type SearchResult struct{
	Results string `json: "Results"`
}

type TorrentData struct {
	Name string
	Id string
	State string
	Progress string
}

const downloadFolder string = "/media2/torrents"
const unknownFolder string = "/media2/finished"
const plexFolder string = "/media2/plex-media"
const dataFile string = "torrent-data.json"

func Start(quit *bool){
	if _, err := os.Stat(dataFile); os.IsNotExist(err) {
		f, err := os.Create(dataFile)
		if err != nil {
			fmt.Println(err.Error())
		}
		defer f.Close()
		torrents := make([]Torrent, 0)
		bytes, _ := json.Marshal(torrents)
		err = ioutil.WriteFile(dataFile, bytes, 0644)
		if err != nil {
			fmt.Println(err.Error())
		}
	}
	for {
		if *quit {
			break
		}
		Purge()
		torrents, _ := GetRunning()
		for _, t := range torrents {
			if t.State == "Error" {
				_, _ = Remove(t.Id)
			} else if t.State == "Seeding" {
				tor, ok := InDataFile(t)
				if ok {
					if tor.TorrentType == "show" {
						if tor.ShowType == "complete" {
							path := plexFolder + "/television/" + tor.Title + "/"
							if _, err := os.Stat(path); os.IsNotExist(err) {
								os.Mkdir(path, 0755)
							}
							season := 1
							filepath.Walk(downloadFolder + "/" + tor.Name, func(p string, info os.FileInfo, err error) error {
								fmt.Println(info.Name())
								if info.IsDir() && info.Name() != tor.Name && strings.Contains(strings.ToLower(info.Name()), "season") {
									seasonPath := path + fmt.Sprintf("Season %02d", season) + "/"
									if _, err := os.Stat(seasonPath); os.IsNotExist(err) {
										os.Mkdir(seasonPath, 0755)
									}
									episode := 1
									fmt.Println(downloadFolder + "/" + tor.Name + "/" + info.Name())
									filepath.Walk(downloadFolder + "/" + tor.Name + "/" + info.Name(), func (p string, in os.FileInfo, err error) error {
										temp := strings.Split(in.Name(), ".")
										ext :=  "." + temp[len(temp) - 1]
										if ext == ".mp4" || ext == ".avi" || ext == ".mkv" {
											fmt.Println(downloadFolder + "/" + tor.Name + "/" + info.Name() + "/" + in.Name())
											fmt.Println(seasonPath + fmt.Sprintf(tor.Title + " - s%02de%02d" + ext, season, episode))
											cmdCommand := "cp"
											cmdArgs := []string{downloadFolder + "/" + tor.Name + "/" + info.Name() + "/" + in.Name(), seasonPath + fmt.Sprintf(tor.Title + " - s%02de%02d" + ext, season, episode)}
											err := exec.Command(cmdCommand, cmdArgs...).Run()
											if err != nil {
												fmt.Println(err.Error())
											}
											episode = episode + 1
										}
										return nil
									})
									season = season + 1
								}
								return nil
							})
						} else if tor.ShowType == "season" {
							path := plexFolder + "/television/" + tor.Title + "/"
							if _, err := os.Stat(path); os.IsNotExist(err) {
								os.Mkdir(path, 0755)
							}
							path = path + fmt.Sprintf("Season %02s", tor.Season) + "/"
							if _, err := os.Stat(path); os.IsNotExist(err) {
								os.Mkdir(path, 0755)
							}
							i := 1
							filepath.Walk(downloadFolder + "/" + tor.Name, func(p string, info os.FileInfo, err error) error {
								temp := strings.Split(info.Name(), ".")
								ext :=  "." + temp[len(temp) - 1]
								if ext == ".mp4" || ext == ".avi" || ext == ".mkv" {
									fmt.Println(downloadFolder + "/" + tor.Name + "/" + info.Name())
									fmt.Println(path + fmt.Sprintf(tor.Title + " - s%02se%02d" + ext, tor.Season, i))
									cmdCommand := "cp"
									cmdArgs := []string{downloadFolder + "/" + tor.Name + "/" + info.Name(), path + fmt.Sprintf(tor.Title + " - s%02se%02d" + ext, tor.Season, i)}
									err := exec.Command(cmdCommand, cmdArgs...).Run()
									if err != nil {
										fmt.Println(err.Error())
									}
									i = i + 1
								}
								return nil
							})
						} else if tor.ShowType == "episode" {
							path := plexFolder + "/television/" + tor.Title + "/"
							if _, err := os.Stat(path); os.IsNotExist(err) {
								os.Mkdir(path, 0755)
							}
							path = path + fmt.Sprintf("Season %02s", tor.Season) + "/"
							if _, err := os.Stat(path); os.IsNotExist(err) {
								os.Mkdir(path, 0755)
							}
							f, err := os.Open(downloadFolder + "/" + tor.Name)
							defer f.Close()
							if err != nil {
								f, _ = os.Open(downloadFolder + "/" + tor.Name + "/")
								defer f.Close()
							}
							stat, err := f.Stat()
							if err != nil {
								fmt.Println(err.Error())
							}
							cmdCommand := "mv"
							var cmdArgs []string
							if stat.IsDir() {
								var file os.FileInfo
								filepath.Walk(downloadFolder + "/" + tor.Name, func(path string, info os.FileInfo, err error) error {
									if !info.IsDir() {
										temp := strings.Split(info.Name(), ".")
										ext :=  "." + temp[len(temp) - 1]
										if file == nil && (ext == ".mp4" || ext == ".avi" || ext == ".mkv") {
											file = info
										}
									}
									return nil
								})
								temp := strings.Split(file.Name(), ".")
								ext :=  "." + temp[len(temp) - 1]
								cmdArgs = []string{downloadFolder + "/" + tor.Name + "/" + file.Name(), path + fmt.Sprintf(tor.Title + " - s%02se%02s" + ext, tor.Season, tor.Episode)}
							} else {
								temp := strings.Split(tor.Name, ".")
								ext :=  "." + temp[len(temp) - 1]
								cmdArgs = []string{downloadFolder + "/" + tor.Name, path + fmt.Sprintf(tor.Title + " - s%02se%02s" + ext, tor.Season, tor.Episode)}
							}
							err = exec.Command(cmdCommand, cmdArgs...).Run()
							if err != nil {
								fmt.Println(err.Error())
							}
						} else if tor.ShowType == "unknown" {
							cmdCommand := "mv"
							cmdArgs := []string{downloadFolder + "/" + t.Name, unknownFolder + "/" + t.Name}
							err := exec.Command(cmdCommand, cmdArgs...).Run()
							if err != nil {
								fmt.Println(err.Error())
							}
						}
					} else if tor.TorrentType == "movie" {
						f, err := os.Open(downloadFolder + "/" + tor.Name)
						defer f.Close()
						if err != nil {
							f, _ = os.Open(downloadFolder + "/" + tor.Name + "/")
							defer f.Close()
						}
						stat, err := f.Stat()
						if err != nil {
							fmt.Println(err.Error())
						}
						if stat.IsDir() {
							var largest os.FileInfo
							filepath.Walk(downloadFolder + "/" + tor.Name, func(path string, info os.FileInfo, err error) error {
								if largest == nil {
									largest = info
								} else {
									if !info.IsDir() {
										temp := strings.Split(info.Name(), ".")
										ext :=  "." + temp[len(temp) - 1]
										if ext == ".mp4" || ext == ".avi" || ext == ".mkv" {
											if info.Size() > largest.Size() { largest = info }
										}
									}
								}
								return nil
							})
							temp := strings.Split(largest.Name(), ".")
							ext := "." + temp[len(temp) - 1]
							cmdCommand := "mv"
							cmdArgs := []string{downloadFolder + "/" + tor.Name + "/" + largest.Name(), plexFolder + "/movies/" + tor.Title + ext}
							err := exec.Command(cmdCommand, cmdArgs...).Run()
							if err != nil {
								fmt.Println(err.Error())
							}
						} else {
							temp := strings.Split(tor.Name, ".")
							ext :=  "." + temp[len(temp) - 1]
							cmdCommand := "mv"
							cmdArgs := []string{downloadFolder + "/" + tor.Name, plexFolder + "/movies/" + tor.Title + ext}
							err := exec.Command(cmdCommand, cmdArgs...).Run()
							if err != nil {
								fmt.Println(err.Error())
							}
						}
					} else if tor.TorrentType == "misc" {
						fmt.Println(tor.User)
						cmdCommand := "mv"
						cmdArgs := []string{downloadFolder + "/" + tor.Name, "/home/" + tor.User + "/ftp/"}
						err := exec.Command(cmdCommand, cmdArgs...).Run()
						if err != nil {
							fmt.Println(err.Error())
						}
					}
				} else {
					cmdCommand := "mv"
					cmdArgs := []string{downloadFolder + "/" + t.Name, unknownFolder + "/" + t.Name}
					err := exec.Command(cmdCommand, cmdArgs...).Run()
					if err != nil {
						fmt.Println(err.Error())
					}
				}
				_, _ = Remove(t.Id)
				Update()
			}
			
		}
		time.Sleep(5 * time.Second)
	}
}

func Update() {
	
}

func Purge() {
	newTorrents := make([]Torrent, 0)
	f, _ := os.Open(dataFile)
	defer f.Close()
	bytes, _ := ioutil.ReadAll(f)
	data := make([]Torrent, 0)
	json.Unmarshal(bytes, &data)
	torrents, _ := GetRunning()
	for _, t := range data {
		in := false
		for _, tor := range torrents {
			if t.Id == tor.Id {
				in = true
				break
			}
		}
		if in {
			newTorrents = append(newTorrents, t)
		}
	}
	bytes, _ = json.Marshal(newTorrents)
	_ = ioutil.WriteFile(dataFile, bytes, 0644)
}

func InDataFile(t TorrentData) (Torrent, bool) {
	var torrent Torrent
	f, _ := os.Open(dataFile)
	defer f.Close()
	bytes, _ := ioutil.ReadAll(f)
	torrents := make([]Torrent, 0)
	json.Unmarshal(bytes, &torrents)
	for _, tor := range torrents {
		if tor.Id == t.Id {
			tor.Name = t.Name
			return tor, true
		}
	}
	return torrent, false
}

func GetRunning() ([]TorrentData, error) {
	cmdCommand := "deluge-console"
	cmdArgs := []string{"info"}
	cmdOut, err := exec.Command(cmdCommand, cmdArgs...).Output()
	if err != nil {
		return nil, err
	}
	data := strings.Split(string(cmdOut), "\n \n")
	if data[0] == "" {
		return make([]TorrentData, 0), nil
	}
	data[0] = data[0][2:]
	torrents := make([]TorrentData, len(data))
	for i, item := range data {
		state := strings.Split(strings.Split(item, "\n")[2], " ")[1]
		var progress string
		if state == "Paused" {
			if strings.Split(item, "\n")[6] == "" {
				progress = "100.00%"
			} else {
				progress = strings.Split(strings.Split(item, "\n")[6], " ")[1]
			}
		} else if state == "Downloading" {
			progress = strings.Split(strings.Split(item, "\n")[7], " ")[1]
		} else {
			progress = "100.00%"
		}
		torrent := TorrentData{
			Name: strings.Join(strings.Split(strings.Split(item, "\n")[0], " ")[1:], " "),
			Id: strings.Split(strings.Split(item, "\n")[1], " ")[1],
			State: state,
			Progress: progress,
		}
		torrents[i] = torrent
	}
	return torrents, nil
}

func Info(id string) (TorrentData, bool, errors.Error) {
	var torrent TorrentData
	torrents, _ := GetRunning()
	for _, torrent = range torrents {
		if torrent.Id == id {
			return torrent, true, errors.Error{}
		}
	}
	return torrent, false, errors.Error{"No torrent matching Id", http.StatusBadRequest}
}

func Pause(id string) (bool, errors.Error) {
	cmdCommand := "deluge-console"
	cmdArgs := []string{"pause", id}
	if err := exec.Command(cmdCommand, cmdArgs...).Run(); err != nil {
		return false, errors.Error{"Error when running command", http.StatusInternalServerError}
	}
	return true, errors.Error{}
}

func Resume(id string) (bool, errors.Error) {
	cmdCommand := "deluge-console"
	cmdArgs := []string{"resume", id}
	if err := exec.Command(cmdCommand, cmdArgs...).Run(); err != nil {
		return false, errors.Error{"Error when running command", http.StatusInternalServerError}
	}
	return true, errors.Error{}
}

func Remove(id string) (bool, errors.Error) {
	cmdCommand := "deluge-console"
	cmdArgs := []string{"rm", "--remove_data", id}
	if err := exec.Command(cmdCommand, cmdArgs...).Run(); err != nil {
		return false, errors.Error{"Error when running command", http.StatusInternalServerError}
	}
	return true, errors.Error{}
}


func Add(t Torrent) (Torrent, bool, errors.Error) {
	if ok, err := t.Validate(); !ok {
		return t, false, err
	}
	cmdCommand := "deluge-console"
	cmdArgs := []string{"add", "-p", downloadFolder, t.Uri}
	cmdOut, err := exec.Command(cmdCommand, cmdArgs...).Output()
	if err != nil {
		return t, false, errors.Error{"Error adding torrent", http.StatusInternalServerError}
	}
	cmdArgs = []string{"info", "--sort=active_time"}
	cmdOut, _ = exec.Command(cmdCommand, cmdArgs...).Output()
	data := strings.Split(string(cmdOut), "\n \n")[0]
	data = data[2:]
	t.Name = strings.Join(strings.Split(strings.Split(data, "\n")[0], " ")[1:], " ")
	t.Id = strings.Split(strings.Split(data, "\n")[1], " ")[1]
	f, err := os.Open(dataFile)
	if err != nil {
		return t, false, errors.Error{"Error opening data file", http.StatusInternalServerError}
	}
	defer f.Close()
	bytes, _ := ioutil.ReadAll(f)
	oldTorrents := make([]Torrent, 0)
	json.Unmarshal(bytes, &oldTorrents)
	torrents := append(oldTorrents, t)
	bytes, _ = json.Marshal(torrents)
	err = ioutil.WriteFile(dataFile, bytes, 0644)
	if err != nil {
		return t, false, errors.Error{"Error saving data file", http.StatusInternalServerError}
	}
	db, err := sql.Open("mysql", "eve:Tightwithay2016$@/eve")
	if err != nil {
		return t, false, errors.Error{"Error accessing database", http.StatusInternalServerError}
	}
	stmt, err := db.Prepare("INSERT torrents SET user=?,title=?,uri=?")
	if err != nil {
		return t, false, errors.Error{"Error accessing database", http.StatusInternalServerError}
	}
	res, err := stmt.Exec(t.UserId, t.Title, t.Uri)
	if err != nil {
		return t, false, errors.Error{"Error adding torrent", http.StatusInternalServerError}
	}
	id, err := res.LastInsertId()
	if err != nil {
		return t, false, errors.Error{"Error accessing database", http.StatusInternalServerError}
	}
	rows, err := db.Query("SELECT total_dl FROM users WHERE id=?", t.UserId)
	if err != nil {
		return t, false, errors.Error{"Error accessing database", http.StatusInternalServerError}
	}
	defer rows.Close()
	var totalDl int
	for rows.Next() {
		err := rows.Scan(&totalDl)
		if err != nil {
			return t, false, errors.Error{"Invalid User", http.StatusBadRequest}
		}
	}
	totalDl = totalDl + 1
	stmt, err = db.Prepare("UPDATE users SET prev_dl=?, total_dl=? WHERE id=?")
	if err != nil {
		return t, false, errors.Error{"Error accessing database", http.StatusInternalServerError}
	}
	_, err = stmt.Exec(id, totalDl, t.UserId)
	if err != nil {
		return t, false, errors.Error{"Error accessing database", http.StatusInternalServerError}
	}
	db.Close()
	return t, true, errors.Error{}
}

func Search(t Torrent) ([]piratebay.Torrent, bool, errors.Error) {
	pb := piratebay.New("https://thepiratebay.org")
	results, err := pb.Search(t.Query)
	if err != nil {
		fmt.Println("asdf")
		return results, false, errors.Error{"Error reaching torrent indexers", http.StatusInternalServerError}
	}
	return results, true, errors.Error{}

	/*
	var results []byte
	url := "http://localhost:9117/torznab/all?apikey=vqk64rcmozdvehw1knsfsrbf6dekge5l&q="
	var jsonStr = []byte(`{"Query": "` + t.Query + `", "Category": "", "Tracker": ""}`)
	fmt.Println(string(jsonStr))
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("asdf")
		return results, false, errors.Error{"Error reaching torrent indexers", http.StatusInternalServerError}
	}
	defer resp.Body.Close()
	results, _ = ioutil.ReadAll(resp.Body)
	return results, true, errors.Error{}
	*/
}

func (t Torrent) Validate() (bool, errors.Error) {
	if t.Uri == "" {
		return false, errors.Error{"No uri specified", http.StatusBadRequest}
	} else if t.TorrentType == "" {
		return false, errors.Error{"Torrent type not specified", http.StatusBadRequest}
	} else if t.TorrentType != "show" && t.TorrentType != "movie" && t.TorrentType != "misc" {
		return false, errors.Error{"Invalid torrent type", http.StatusBadRequest}
	}
	if t.TorrentType == "show" {
		if t.Title == "" {
			return false, errors.Error{"No title specified", http.StatusBadRequest}
		} else if t.ShowType == "" {
			return false, errors.Error{"No show type specified", http.StatusBadRequest}
		} else if t.ShowType != "season" && t.ShowType != "episode" && t.ShowType != "complete" && t.ShowType != "unknown" {
			return false, errors.Error{"Invalid show type", http.StatusBadRequest}
		}
		if t.ShowType == "season" && t.Season == "" {
			return false, errors.Error{"No season specified", http.StatusBadRequest}
		} else if t.ShowType == "episode" {
			if t.Season == "" {
				return false, errors.Error{"No season specified", http.StatusBadRequest}
			} else if t.Episode == "" {
				return false, errors.Error{"No episode specified", http.StatusBadRequest}
			}
		}
	} else if t.TorrentType == "movie" {
		if t.Title == "" {
			return false, errors.Error{"No title specified", http.StatusBadRequest}
		}
	} else if t.TorrentType == "misc" {
		if t.User == "" {
			return false, errors.Error{"No user specified", http.StatusBadRequest}
		}
	}
	return true, errors.Error{}
}
