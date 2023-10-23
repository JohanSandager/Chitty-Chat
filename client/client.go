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

// Global variables
var sAddress = flag.String("sAddress", "", "Server address")
var user_name = flag.String("usr", "", "User name")
var client_lamport_timestamp = 0

func main() {
	var wg sync.WaitGroup
	wg.Add(1)

	flag.Parse()

	connection := ConnectToServer(*sAddress)
	client := GetClientFromConnection(connection)
	stream := GetStream(client)
	go Chat(*user_name, &wg, stream)

	wg.Wait()
}

// Dials the server and returns a connection.
func ConnectToServer(server_address string) *grpc.ClientConn {
	IncrementAndPrintLamportTimestamp("Dial server")
	connection, err := grpc.Dial(server_address, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("Could not connect to %v", server_address)
	}

	log.Printf("Connected to the server at %v\n", server_address)

	return connection
}

// Creates and returns a ChitChat Client from a grpc client connection.
func GetClientFromConnection(connection *grpc.ClientConn) pb.ChitChatClient {
	client := pb.NewChitChatClient(connection)
	return client
}

/*
Tells the server that it is ready to chat by sending a connection request.
It starts two go routines one for listening and one for sending messages
*/
func Chat(user_name string, wg *sync.WaitGroup, stream pb.ChitChat_ChatClient) {
	SendConnectionRequest(user_name, stream)

	go RecieveMessages(stream)
	go SendMessage(stream, user_name, wg)
}

/*
Will continiously listen for messages and print them while the stream remains open.
If the stream terminates ungracefully it will throw an error, otherwise it will just return.
*/
func RecieveMessages(stream pb.ChitChat_ChatClient) {
	for {

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

		PrintMessage(inbound_message.GetMessage())

		SetAndPrintLamportTimestamp("Validate recieved timestamp", ValidateLamportTimestamp(client_lamport_timestamp, int(inbound_message.GetLamportTimestamp()))+1)
	}
}

// Listens for user input, upon recieving such it will determaine wheter it should close the connection, accept the message and send it or throw an error.
func SendMessage(stream pb.ChitChat_ChatClient, user_name string, wg *sync.WaitGroup) {
	defer wg.Done()

	scanner := bufio.NewScanner(os.Stdin)

	for scanner.Scan() {
		if input := scanner.Text(); input == "c" {
			SendDisconnectionRequest(user_name, stream)
			time.Sleep(time.Millisecond * 100) // Sleep allows the resources to be freed before they are terminated by the return
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

// Invokes the RPC returning a stream.
func GetStream(client pb.ChitChatClient) pb.ChitChat_ChatClient {
	stream, err := client.Chat(context.Background())

	if err != nil {
		log.Fatalf("Could call gRPC: %v", err)
	}

	return stream
}

// Publishes a message to the stream.
func Publish(stream pb.ChitChat_ChatClient, user_name string, input string) {
	IncrementAndPrintLamportTimestamp("Publish message")
	message := CreateChitChatMessageObject(user_name, input)
	stream.Send(message)
}

// Logs a formatted ChitChat message.
func PrintMessage(message *pb.ChitChatMessage) {
	log.Printf(message.GetUserName() + ": " + message.GetMessage())
}

// Sends a connection request.
func SendConnectionRequest(user_name string, stream pb.ChitChat_ChatClient) {
	IncrementAndPrintLamportTimestamp("Send connection request")
	message := CreateConnectionRequestObject(user_name)
	if err := stream.Send(message); err != nil {
		log.Fatalf("Failed to send connection request: %v", err)
	}
}

// Sends a disconnection request.
func SendDisconnectionRequest(user_name string, stream pb.ChitChat_ChatClient) {
	message := CreateDisconnectionRequestObject(user_name)
	if err := stream.Send(message); err != nil {
		log.Fatalf("Failed to send disconnection request: %v", err)
	}

	if err := stream.CloseSend(); err != nil {
		log.Fatalf("Failed to close the send stream: %v", err)
	}
}

// Creates a ChitChat message rapped in a message container.
func CreateChitChatMessageObject(user_name string, input string) *pb.ChitChatInformationContainer {
	outbound_message := &pb.ChitChatMessage{UserName: user_name, Message: input}
	message_container := &pb.ChitChatInformationContainer{
		LamportTimestamp: int64(client_lamport_timestamp),
		These:            &pb.ChitChatInformationContainer_Message{Message: outbound_message},
	}
	return message_container
}

// Creates a disconnection request rapped in a message container.
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

// Creates a connection request rapped in a message container.
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

// Validates wheter the client of the sever Lampert timestamp is the correct one.
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
