package main

import (
	"bufio"
	pb "chat_service/grpc"
	"context"
	"flag"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var sPort = flag.String("sPort", "", "Server port")
var user_name = flag.String("usr", "", "User name")

func main() {
	var wg sync.WaitGroup
	wg.Add(1)

	flag.Parse()

	go Chat(*sPort, *user_name, &wg)

	wg.Wait()
}

func ConnectToServer(server_address string) (pb.ChitChatClient, error) {
	connection, err := grpc.Dial(server_address, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("Could not connect to %v", server_address)
	}

	log.Printf("Connected to the server at %v\n", server_address)

	return pb.NewChitChatClient(connection), nil
}

func Chat(server_address string, user_name string, wg *sync.WaitGroup) {
	server_connection, _ := ConnectToServer(server_address)

	ctx, cancel := context.WithCancel(context.Background())

	stream, _ := server_connection.Chat(ctx)

	outbound_message := &pb.ConnectionRequest{
		UserName: user_name,
	}
	message_container := &pb.ChitChatInformationContainer{
		These: &pb.ChitChatInformationContainer_ConnectionRequest{
			ConnectionRequest: outbound_message,
		},
	}

	stream.Send(message_container)

	go RecieveMessages(stream)
	go SendMessage(stream, user_name, wg, cancel)
}

func RecieveMessages(stream pb.ChitChat_ChatClient) {
	for {
		inbound_message, err := stream.Recv()

		if err == io.EOF {
			return
		} else if err != nil {
			log.Fatalf("Failed: %v", err)
		}

		log.Printf(inbound_message.GetMessage().GetUserName() + ": " + inbound_message.GetMessage().GetMessage())
	}
}

func SendMessage(stream pb.ChitChat_ChatClient, user_name string, wg *sync.WaitGroup, cancel context.CancelFunc) {
	defer wg.Done()

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		input := scanner.Text()

		if input == "c" {
			outbound_message := &pb.DisconnectionRequest{
				UserName: user_name,
			}
			message_container := &pb.ChitChatInformationContainer{
				These: &pb.ChitChatInformationContainer_DisconnectionRequest{DisconnectionRequest: outbound_message},
			}

			stream.Send(message_container)
			stream.CloseSend()
			time.Sleep(time.Millisecond * 100)
			return
		}

		outbound_message := &pb.ChitChatMessage{UserName: user_name, Message: input}
		message_container := &pb.ChitChatInformationContainer{
			These: &pb.ChitChatInformationContainer_Message{Message: outbound_message},
		}

		stream.Send(message_container)
	}

	if err := stream.CloseSend(); err != nil {
		log.Fatalf("Failed to close the send stream: %v", err)
	}
}
