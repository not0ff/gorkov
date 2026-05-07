# gorkov

*Gorkov* - a discord chatbot implemented using markov chains in golang

## Installation & usage

- Install and run directly 
```
$ go install -ldflags="-s -w" github.com/not0ff/gorkov@latest
$ gorkov -token <token> -guildIDs <guild_1>,<guild_2>
```

- Clone the repo and run using docker compose
```
$ git clone github.com/not0ff/gorkov
$ TOKEN=<token> GUILDS=<guild_1>,<guild_2> docker compose up -d --build
```

*The bot needs to receive chat messages to learn from before generating the first output


## Features
- [x] Learning from chat messages
- [x] Randomly responding to users
- [x] Slash commands for direct control
- [ ] Configuration workflow
- [x] Award and reinforce output using reactions

## License
Licensed under GPLv3