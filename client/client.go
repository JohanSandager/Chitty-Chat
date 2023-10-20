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

	ctx, cancel := context.WithCancel(context.Background())

	stream, _ := server_connection.Chat(ctx)

	IncrementAndPrintLamportTimestamp("Send connection request")

	outbound_message := &pb.ConnectionRequest{
		UserName: user_name,
	}
	message_container := &pb.ChitChatInformationContainer{
		LamportTimestamp: int64(client_lamport_timestamp),
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
		IncrementAndPrintLamportTimestamp("Recieve message")

		inbound_message, err := stream.Recv()

		if err == io.EOF {
			return
		} else if status, ok := status.FromError(err); ok {
			if status.Code() == codes.Unavailable {
				log.Fatalf("Terminated ungracefully: %v", status.Message())
			}
		} else if err != nil {
			log.Fatalf("Failed: %v", err)
		}

		SetAndPrintLamportTimestamp("Validate timestamp", ValidateLamportTimestamp(client_lamport_timestamp, int(inbound_message.GetLamportTimestamp())))

		log.Printf(inbound_message.GetMessage().GetUserName() + ": " + inbound_message.GetMessage().GetMessage())
	}
}

func SendMessage(stream pb.ChitChat_ChatClient, user_name string, wg *sync.WaitGroup, cancel context.CancelFunc) {
	defer wg.Done()

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		input := scanner.Text()

		IncrementAndPrintLamportTimestamp("Publish message")

		if input == "c" {
			outbound_message := &pb.DisconnectionRequest{
				UserName: user_name,
			}
			message_container := &pb.ChitChatInformationContainer{
				LamportTimestamp: int64(client_lamport_timestamp),
				These:            &pb.ChitChatInformationContainer_DisconnectionRequest{DisconnectionRequest: outbound_message},
			}

			stream.Send(message_container)
			stream.CloseSend()
			time.Sleep(time.Millisecond * 100)
			return
		}

		outbound_message := &pb.ChitChatMessage{UserName: user_name, Message: input}
		message_container := &pb.ChitChatInformationContainer{
			LamportTimestamp: int64(client_lamport_timestamp + 1),
			These:            &pb.ChitChatInformationContainer_Message{Message: outbound_message},
		}

		stream.Send(message_container)
	}

	if err := stream.CloseSend(); err != nil {
		log.Fatalf("Failed to close the send stream: %v", err)
	}
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
