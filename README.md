# Go SMPP REST implementation

An *extremely* simple REST Server Implementing [gosmpp](https://github.com/linxGnu/gosmpp) client library.

## About
- Got to know about SMPP(Short Message Peer-to-Peer Protocol) and all its intricacies.
- Further my knowledge of Go.

## Modules

- esme - the implementation of an External Short Message Entity (ESME) client
- smsc - a simple SMSC simulator (Picked directly from gosmpp [example smsc](https://github.com/linxGnu/gosmpp/tree/master/example/smsc))


## Usage

1. SMSC (Optional)
   - Navigate to SMSC directory
   - Build & Run SMSC (g++ required): `./run.sh`

2. ESME REST api
   - Navigate to project directory
   - Build: `go build` and then run `./gosmpp-api`
   - Or combine and run `go run .`


### Api Endpoints
1. `http://localhost:8081/send` - POST
   - Send a request with message in body
   - You should get a similar response to 
     - `{
       "id": 42825,
       "message": "Hello World!",
       "status": "PENDING"
       }`


2. `http://localhost:8081/status/29`
   - Send a query with id returned by send endpoint
   - You should get a similar response as below: (note the status change)
      - `{
        "id": 42825,
        "message": "Hello World!",
        "status": "DELIVRD"
        }`


2. `http://localhost:8081/all`
   - Send a query to retrieve all messages on the server
   

## Improvements
- There seems to be duplication with the DeliverSM response from either the SMSC or the library. Fortunately, it does not really affect the running of the servers.
- We could add a permanent datastore, currently it only saves to memory and is gone when server is restarted.
- We could add webhook functionality to inform a REST client of a status change rather than them querying for updates.
- The mutex Locks, I believe, could be done better.
- There seems to be a library (https://github.com/fiorix/go-smpp) that we could benchmark gosmpp against and see the pros and cons.
- High number of requests fail and can be reduced by SMPP v5.0 with the use of `congestion_state` TLV. So, an upgrade to the lib would be advantageous.

#### Feedback/Contribution
Is always welcome. :)