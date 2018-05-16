# Nakkiservo slack-nuke

## Usage

```
  make
  ./slack-nuke -api_key <your key here> -target <channel name>
```

## OR if you want, just run this in bash instead

```
curl -s "https://slack.com/api/files.list?token=<your_token>&count=1000&page=1&pretty=1" | grep '"id":' | awk -F'"' '{print $4}' | xargs printf "curl -s 'https://slack.com/api/files.delete?token=<your_token>&file=%s&pretty=1'" | bash
```



## TODO: 

  - Ability to shotgun self (delete own files)
  - Token as a flag or something
  - Coffee making capabilities (IMPORTANT)
