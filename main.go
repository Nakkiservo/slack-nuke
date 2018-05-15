package main

import (
  "os"
  "fmt"
  "net/http"
  "encoding/json"
  "time"
  "bytes"
)

const TOKEN = "your token here"
const NUM_WORKERS = 10
const RATE_LIMIT_SECONDS = 2

func main() {
  fmt.Println("NUKING ALL THE THINGS!")

  if len(os.Args) < 2 {
    fmt.Println("Usage: slack-nuke <channel name>")
    return
  }

  target := os.Args[1]
  fmt.Println("Target: #", target)

  channels, err := GetChannelList()

  if err != nil {
    panic(err)
  }

  fmt.Printf("Locating target channel....")
  var channel_id string

  if list, ok := channels["channels"]; ok {
    chanList := list.([]interface{})
    for _, v := range chanList {
      chanInfo := v.(map[string]interface{})
      if chanInfo["name"] == target {
        fmt.Print("FOUND!\n")
        channel_id = chanInfo["id"].(string)
        break
      }
    }
  } else {
    fmt.Println("No channels returned")
  }

  if channel_id == "" {
    fmt.Printf(" unable to find channel #%s\n", target)
    return
  }

  if fileList, err := GetFileList(channel_id); err == nil {
    if len(fileList) == 0 {
      fmt.Println("No files to delete...")
      return
    }
    DeleteFiles(fileList)
  } else {
    fmt.Println("Unable to get files: " + err.Error())
  }
}

func GetChannelList() (map[string]interface{}, error) {
  apiUrl := "https://slack.com/api/channels.list"

  req, err := http.NewRequest("GET", apiUrl, nil)

  if err != nil {
    return nil, err
  }

  q := req.URL.Query()
  q.Add("token", TOKEN)

  req.URL.RawQuery = q.Encode()

  client := http.Client{}
  resp, err := client.Do(req)

  files := make(map[string]interface{})

  if err = json.NewDecoder(resp.Body).Decode(&files); err != nil {
    return nil, err
  }

  return files, nil
}

// GetFileList returns a list of file ids for given channel id
func GetFileList(channel_id string) ([]string, error) {
  apiUrl := "https://slack.com/api/files.list"

  req, err := http.NewRequest("GET", apiUrl, nil)

  if err != nil {
    return nil, err
  }

  q := req.URL.Query()
  q.Add("token", TOKEN)
  q.Add("channel", channel_id)
  q.Add("count", fmt.Sprintf("%d", 200)) // 200 items per list, for now...

  req.URL.RawQuery = q.Encode()

  client := http.Client{}
  resp, err := client.Do(req)

  fileResp := make(map[string]interface{})

  if err = json.NewDecoder(resp.Body).Decode(&fileResp); err != nil {
    return nil, err
  }

  if _, ok := fileResp["ok"]; ok {
    files, _ := fileResp["files"].([]interface{})
    fileList := make([]string, len(files), len(files))
    count := 0
    for i, f := range files {
      file := f.(map[string]interface{})
      if id, ok := file["id"]; ok {
        fileList[i] = id.(string)
        count++
      }
    }
    fmt.Printf("%d files found.\n", count)
    return fileList, nil
  }

  return nil, fmt.Errorf("Unable to get file list.")
}

func DeleteFiles(file_ids []string) {
  jobs := make(chan string, len(file_ids))
  results := make(chan error, len(file_ids))
  fmt.Printf("%d jobs\n", len(file_ids))
  fmt.Println("Starting workers.")

  for w := 1; w <= NUM_WORKERS; w++ {
    go DeleteWorker(w, jobs, results)
  }

  for i := 0; i < len(file_ids); i++ {
    jobs <- file_ids[i]
  }

  close(jobs)

  for i := 0; i < len(file_ids); i++ {
    <-results
  }
}

func DeleteWorker(id int, jobs <-chan string, results chan<- error) {
  //apiUrl := "https://slack.com/api/files.delete"
  fmt.Printf("Starting worker ID %d\n", id)

  for job := range jobs {
    fmt.Printf("Worker %d: deleting file id %s\n", id, job)
    err := doDelete(job)
    if err != nil {
      fmt.Printf("Worker %d: unable to delete file id '%s': %s\n", id, job, err.Error())
    }
    time.Sleep(time.Second * time.Duration(RATE_LIMIT_SECONDS))
    results <- err
  }
}

func doDelete(file_id string) error {
  apiUrl := "https://slack.com/api/files.delete"

  data := make(map[string]interface{})
  data["file"] = file_id

  b := new (bytes.Buffer)
  json.NewEncoder(b).Encode(&data)

  r, _ := http.NewRequest("POST", apiUrl, b)
  r.Header.Add("Content-Type", "application/json")
  r.Header.Add("Authorization", "Bearer " + TOKEN)

  client := http.Client{}

  resp, err := client.Do(r)
  if err != nil {
    resp.Body.Close()
    return err
  }

  responseData := make(map[string]interface{})

  json.NewDecoder(resp.Body).Decode(&responseData)

  if okResp, ok := responseData["ok"]; ok {
    k := okResp.(bool)
    if !k {
      fmt.Println("Response is false!")
      fmt.Println(responseData)
      return fmt.Errorf("Response not ok!")
    }
  } else {
    return fmt.Errorf("Erroneous response, yeah yeah")
  }

  return nil
}
