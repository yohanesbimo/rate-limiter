How to run this application:
1. Install go on your machine
2. Install redis on your machine
3. run `go run main.go` at the root directory of this application
4. http server ready to serve on your machine at `http://localhost:3000/`

How to make sure this application run well:
1. open your terminal
2. go to this  application root directory
3. run `./test.sh`
4. check result at this root application directory, named `first.txt` and `second.txt`

Overview:
This is simple HTTP server with rate limiter middleware. I build the limiter using Redis.
Just add this middleware for each endpoint API that need to be rate limit.
The function will check how many request per minute and check if client (marked with API key) elligible to process the request or not.
This middleware function can be easily added to any endpoint and this middleware function built for gin framework.


Algorithm:
1. Lock client (marked by API key) when request is coming.
2. If another client request still running, then latest client request will return error.
3. After client request success to Lock, check if client have remaining balance to process request.
4. Client will receive 429 HTTP status code if no remaining balance to process request.
5. When client elligible to process request, this middleware function will add request counter for this client and save to Redis.

Assumption or Limitation:
This rate limiter middleware still only save configuration in local variable for each client.
It can be moved to database so you can more flexible to set configuration for each user.
This middleware only check limit for every minute. It can be added (eg. every 5 minutes, every hour) with some configuration on database.