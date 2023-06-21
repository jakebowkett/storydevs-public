# storydevs

[![Actions Status](https://github.com/jakebowkett/storydevs/workflows/Go/badge.svg)](https://github.com/jakebowkett/storydevs/actions)

Copyright (C) Jake Bowkett - All Rights Reserved
Unauthorized copying of this program or parts thereof, via any medium is strictly prohibited
Proprietary and confidential
Written by Jake Bowkett <jake.bowkett01@gmail.com>, 2023

---

# Running Locally

1. Copy `config.local.example.toml` and name it `config.local.toml`.
2. Copy `credentials.example.toml` and name it `credentials.local.toml`.
3. A Postgres service must be running locally on port 5432. Make sure the Postgres service is using the same settings specified in `credentials.local.toml` under `DbConn`.
4. Ensure version of Go referenced in `go.mod` is installed.
5. Run `go build` in `/cmd/server` and run the resulting executable to start the server.
6. Visit `localhost:3030` to see the site.