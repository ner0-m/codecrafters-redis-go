[![progress-banner](https://backend.codecrafters.io/progress/redis/041cb111-5b3a-45f3-bc18-082b503c8005)](https://app.codecrafters.io/users/ner0-m?r=2qF)

This is my implementation of ["Build Your Own Redis"
Challenge](https://codecrafters.io/challenges/redis) using Go.

In this challenge, portions of the Redis protocol are implemented. It's a real fun
experience, and I used it both as a learning experience for Go and learning a
bit more about implementing protocols, handling messages and everything
involved!

## Trying it out

1. Clone the repo
1. Ensure you have `go (1.19)` installed locally (I'm using version `1.22`)
1. Run `./spawn_redis_server.sh` to run your Redis server
1. Use `redis-cli` to interact with the server (e.g. `redis-cli echo hello`)
