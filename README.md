<!-- GitAds-Verify: 3GO71XTDXU9WLKO8E4AGFKS62M1GRFJC -->

## What is this?

This is a simple Golang script that uses CloudFlare's API to updates the A records IP address of a domain to the current public IP address of the machine running the script.

## Why did I make this?

I made this script because I needed to needed to update the A records of a domain I own that uses CloudFlare's DNS service. My ISP gave me a dynamic IP address and I needed a tool that would speed up this process.

## How do I use this?

### From source
1. Clone the repository
2. Update the `config.yaml` @ `./CONFIG/config.yaml` file with your CloudFlare API key and email.
3. If you want to create a new A record for a domain, uncomment the /config/create.yaml file and fill in the fields with the correct information, and once ran recomment out the line so it does not run again.
3. Run the script with `go run main.go` or `go build main.go` and then run the executable.

### From Docker
1. Copy the Docker-compose file to your machine
2. copy and update the `config.yaml` file with your CloudFlare API key and email.
3. update the `docker-compose.yaml` file with the correct path to the `config.yaml` file and if logs are wanted add a path that is in the `/config` directory.

## How Do I get my CloudFlare API key?

Follow cloudflare's documentation on how to get your API key [here](https://developers.cloudflare.com/fundamentals/api/get-started/create-token/).

## Can I contribute?

Yes! Feel free to fork the repository and make a pull request.


### Source
<a src =https://developers.cloudflare.com/api> CloudFlare's API Documentation </a>


## What Docker image do I use?

look at https://hub.docker.com/r/auswar/cloudflare_dns_update

