package main

import (
	"bufio"
	pb "chat_service/grpc"
	"context"
	"flag"
	"io"
	"log"
	"math"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

var sPort = flag.String("sPort", "", "Server port")
var user_name = flag.String("usr", "", "User name")
var client_lamport_timestamp = 0

func main() {
	var wg sync.WaitGroup
	wg.Add(1)

	flag.Parse()

	go Chat(*sPort, *user_name, &wg)

	wg.Wait()
}

func ConnectToServer(server_address string) (pb.ChitChatClient, error) {
	IncrementAndPrintLamportTimestamp("Dial server")
	connection, err := grpc.Dial(server_address, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("Could not connect to %v", server_address)
	}

	log.Printf("Connected to the server at %v\n", server_address)

	return pb.NewChitChatClient(connection), nil
}

func Chat(server_address string, user_name string, wg *sync.WaitGroup) {
	server_connection, _ := ConnectToServer(server_address)

	stream, _ := server_connection.Chat(context.Background())

	SendConnectionRequest(user_name, stream)

	go RecieveMessages(stream)
	go SendMessage(stream, user_name, wg)
}

func RecieveMessages(stream pb.ChitChat_ChatClient) {
	for {
		IncrementAndPrintLamportTimestamp("Recieve message")

		inbound_message, err := stream.Recv()

		if err == io.EOF {
			return
		} else if status, ok := status.FromError(err); ok { // This error handeling was helped by ChatGPT
			if status.Code() == codes.Unavailable {
				log.Fatalf("Terminated ungracefully: %v", status.Message())
			}
		} else if err != nil {
			log.Fatalf("Failed: %v", err)
		}

		SetAndPrintLamportTimestamp("Validate timestamp", ValidateLamportTimestamp(client_lamport_timestamp, int(inbound_message.GetLamportTimestamp())))

		PrintMessage(inbound_message.GetMessage())
	}
}

func SendMessage(stream pb.ChitChat_ChatClient, user_name string, wg *sync.WaitGroup) {
	defer wg.Done()

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		if input := scanner.Text(); input == "c" {
			SendDisconnectionRequest(user_name, stream)
			return
		} else if len(input) <= 128 {
			Publish(stream, user_name, input)
		} else if len(input) > 128 {
			log.Printf("Message length expected under 128 characters, actual: %v", len(input))
		} else {
			log.Fatal("Something went wrong")
		}
	}
}

func Publish(stream pb.ChitChat_ChatClient, user_name string, input string) {
	IncrementAndPrintLamportTimestamp("Publish message")
	message := CreateChitChatMessageObject(user_name, input)
	stream.Send(message)
}

func PrintMessage(message *pb.ChitChatMessage) {
	log.Printf(message.GetUserName() + ": " + message.GetMessage())
}

func SendConnectionRequest(user_name string, stream pb.ChitChat_ChatClient) {
	IncrementAndPrintLamportTimestamp("Send connection request")
	message := CreateConnectionRequestObject(user_name)
	if err := stream.Send(message); err != nil {
		log.Fatalf("Failed to send connection request: %v", err)
	}
}

func SendDisconnectionRequest(user_name string, stream pb.ChitChat_ChatClient) {
	message := CreateDisconnectionRequestObject(user_name)
	if err := stream.Send(message); err != nil {
		log.Fatalf("Failed to send disconnection request: %v", err)
	}

	if err := stream.CloseSend(); err != nil {
		log.Fatalf("Failed to close the send stream: %v", err)
	}

	time.Sleep(time.Millisecond * 100)
}

func CreateChitChatMessageObject(user_name string, input string) *pb.ChitChatInformationContainer {
	outbound_message := &pb.ChitChatMessage{UserName: user_name, Message: input}
	message_container := &pb.ChitChatInformationContainer{
		LamportTimestamp: int64(client_lamport_timestamp + 1),
		These:            &pb.ChitChatInformationContainer_Message{Message: outbound_message},
	}
	return message_container
}

func CreateDisconnectionRequestObject(user_name string) *pb.ChitChatInformationContainer {
	outbound_message := &pb.DisconnectionRequest{
		UserName: user_name,
	}
	message_container := &pb.ChitChatInformationContainer{
		LamportTimestamp: int64(client_lamport_timestamp),
		These:            &pb.ChitChatInformationContainer_DisconnectionRequest{DisconnectionRequest: outbound_message},
	}
	return message_container
}

func CreateConnectionRequestObject(user_name string) *pb.ChitChatInformationContainer {
	outbound_message := &pb.ConnectionRequest{
		UserName: user_name,
	}
	message_container := &pb.ChitChatInformationContainer{
		LamportTimestamp: int64(client_lamport_timestamp),
		These: &pb.ChitChatInformationContainer_ConnectionRequest{
			ConnectionRequest: outbound_message,
		},
	}
	return message_container
}

func ValidateLamportTimestamp(client_timestamp int, server_timestamp int) int {
	return int(math.Max(float64(client_timestamp), float64(server_timestamp)))
}

func IncrementAndPrintLamportTimestamp(action string) {
	client_lamport_timestamp++
	log.Printf("%v has incremented Lamport timestamp to: %v", action, client_lamport_timestamp)
}

func SetAndPrintLamportTimestamp(action string, new_value int) {
	client_lamport_timestamp = new_value
	log.Printf("%v has set Lamport timestamp to: %v", action, client_lamport_timestamp)
}
