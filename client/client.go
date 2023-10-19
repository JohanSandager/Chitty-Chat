package main

import (
	"bufio"
	proto "chat_service/grpc"
	"context"
	"flag"
	"io"
	"log"
	"os"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var sPort = flag.String("sPort", "", "Server port")
var user_name = flag.String("usr", "", "User name")

func main() {
	var wg sync.WaitGroup
	wg.Add(1)

	flag.Parse()

	go chat(*sPort, *user_name, &wg)

	wg.Wait()
}

func connectToServer(serverAddress string) (proto.ChitChatClient, error) {
	conn, err := grpc.Dial(serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("Could not connect to port %v", serverAddress)
	}
	log.Printf("Connected to the server at port %v\n", serverAddress)
	return proto.NewChitChatClient(conn), nil
}

func chat(serverAddress string, user_name string, wg *sync.WaitGroup) {
	serverConnection, _ := connectToServer(serverAddress)

	stream, _ := serverConnection.Chat(context.Background())

	outbound_message := &proto.ConnectionRequest{
		UserName: user_name,
	}
	container := &proto.ChitChatInformationContainer{
		These: &proto.ChitChatInformationContainer_ConnectionRequest{ConnectionRequest: outbound_message},
	}

	stream.Send(container)

	go recieveMessages(stream)
	go sendMessage(stream, user_name, wg)
}

func recieveMessages(stream proto.ChitChat_ChatClient) {
	for {
		inbound_message, err := stream.Recv()
		if err == io.EOF {
			return
		} else if err != nil {
			log.Fatalf("Failed: %v", err)
		}
		log.Printf(inbound_message.GetUserName() + ": " + inbound_message.GetMessage())
	}
}

func sendMessage(stream proto.ChitChat_ChatClient, user_name string, wg *sync.WaitGroup) {
	defer wg.Done()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		input := scanner.Text()

		if input == "close connection" {
			return
		}

		outbound_message := &proto.ChitChatMessage{LamportTimestamp: 0, UserName: user_name, Message: input}
		container := &proto.ChitChatInformationContainer{
			These: &proto.ChitChatInformationContainer_Message{Message: outbound_message},
		}

		stream.Send(container)
	}
}
