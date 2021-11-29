package main

import (
	"context"
	"encoding/json"
	"fmt"
	esmeServer "github.com/drh97/gosmpp-api/esme"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var esme *esmeServer.Esme

func startEsmeServer() {
	x, err := esmeServer.StartSession()
	if err != nil {
		fmt.Println(err)
		panic(err)
	}

	esme = x
}

func startHttpServer() {
	mux := http.NewServeMux()

	//This was used to test a high number of requests and race conditions
	//for i := 0; i < 10000; i++ {
	//	message := esmeServer.ShortMessage{
	//		Message: strconv.Itoa(i),
	//	}
	//	esme.SendSM(&message)
	//}

	mux.HandleFunc("/send", func(w http.ResponseWriter, r *http.Request) {
		message := esmeServer.ShortMessage{}

		err := decodeJSONBody(w, r, &message)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		esme.SendSM(&message)

		b, mErr := json.Marshal(message)
		if mErr != nil {
			fmt.Println(err)
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	})

	mux.HandleFunc("/status/", func(w http.ResponseWriter, r *http.Request) {
		rawId := r.URL.Path[len("/status/"):]

		id, _ := strconv.ParseInt(rawId, 10, 32)

		message := esme.FindMessageBySequence(int32(id))

		b, _ := json.Marshal(message)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	})

	mux.HandleFunc("/all", func(w http.ResponseWriter, r *http.Request) {

		b, _ := json.Marshal(esme.GetMessages())

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(b)
	})

	srv := &http.Server{
		Addr:    ":8081",
		Handler: mux,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	log.Print("Server Started")

	<-done
	log.Print("Server Stopped")

	esme.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
	log.Print("Server Exited Properly")
}

func main() {
	startEsmeServer()
	defer esme.Close()

	startHttpServer()
}
